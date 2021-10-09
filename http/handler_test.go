package http

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"go.uber.org/zap"

	"cabourotte/healthcheck"
	"cabourotte/memorystore"
	"cabourotte/prometheus"
)

func TestHandlers(t *testing.T) {
	prom, err := prometheus.New()
	if err != nil {
		t.Fatalf("Error creating prometheus component :\n%v", err)
	}
	logger := zap.NewExample()
	memstore := memorystore.NewMemoryStore(logger)
	healthcheck, err := healthcheck.New(zap.NewExample(), make(chan *healthcheck.Result, 10), prom)
	if err != nil {
		t.Fatalf("Fail to create the healthcheck component\n%v", err)
	}
	component, err := New(logger, memstore, prom, &Configuration{Host: "127.0.0.1", Port: 2001}, healthcheck)
	if err != nil {
		t.Fatalf("Fail to create the component\n%v", err)
	}
	err = component.Start()
	if err != nil {
		t.Fatalf("Fail to start the component\n%v", err)
	}
	cases := []struct {
		endpoint string
		payload  string
	}{
		{
			endpoint: "/healthcheck/dns",
			payload:  `{"name":"foo","description":"bar","domain":"mcorbin.fr","interval":"10m","one-off":false}`,
		},
		{
			endpoint: "/healthcheck/tcp",
			payload:  `{"name":"bar","description":"bar","interval":"10m","one-off":false,"target":"mcorbin.fr","port":9999,"timeout":"10s"}`,
		},
		{
			endpoint: "/healthcheck/http",
			payload:  `{"name":"baz","description":"bar","domain":"mcorbin.fr","interval":"10m","one-off":false,"target":"mcorbin.fr","port":9999,"timeout":"10s","protocol":"http","valid-status":[200]}`,
		},
		{
			endpoint: "/healthcheck/tls",
			payload:  `{"name":"tls-check","description":"bar","interval":"10m","one-off":false,"target":"mcorbin.fr","port":9999,"timeout":"10s"}`,
		},
	}
	client := &http.Client{}
	for _, c := range cases {
		req, err := http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:2001%s", c.endpoint), bytes.NewBuffer([]byte(c.payload)))
		if err != nil {
			t.Fatalf("Fail to build the HTTP request\n%v", err)
		}
		req.Header.Set("Content-Type", "application/json")
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("HTTP request failed\n%v", err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Fatalf("HTTP request failed, status %d", resp.StatusCode)
		}
	}
	if len(healthcheck.Healthchecks) != 4 {
		t.Fatalf("Healthchecks were not successfully created: %d", len(healthcheck.Healthchecks))
	}

	// get the healthchecks
	resp, err := http.Get("http://127.0.0.1:2001/healthcheck")
	if err != nil {
		t.Fatalf("Fail to get the healthchecks\n%v", err)
	}
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Fail to read the body\n%v", err)
	}
	body := string(bodyBytes)
	if !strings.Contains(body, `"name":"foo"`) {
		t.Fatalf("Invalid body\n")
	}
	if !strings.Contains(body, `"name":"bar"`) {
		t.Fatalf("Invalid body\n")
	}
	if !strings.Contains(body, `"name":"baz"`) {
		t.Fatalf("Invalid body\n")
	}
	// get one healthcheck
	resp, err = http.Get("http://127.0.0.1:2001/healthcheck/foo")
	if err != nil {
		t.Fatalf("Fail to get the healthchecks\n%v", err)
	}
	defer resp.Body.Close()
	bodyBytes, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Fail to read the body\n%v", err)
	}
	body = string(bodyBytes)
	if !strings.Contains(body, `"name":"foo"`) {
		t.Fatalf("Invalid body\n")
	}
	// get one invalid healthcheck
	resp, err = http.Get("http://127.0.0.1:2001/healthcheck/doesnotexist")
	if err != nil {
		t.Fatalf("Fail to get the healthchecks\n%v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNotFound {
		t.Fatalf("Was expecting a 404 response, got %d", resp.StatusCode)
	}
	bodyBytes, err = ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Fail to read the body\n%v", err)
	}
	body = string(bodyBytes)
	if !strings.Contains(body, `not found`) {
		t.Fatalf("Invalid body\n")
	}
	// delete everything
	checks := []string{"foo", "bar", "baz", "tls-check"}
	for _, c := range checks {
		req, err := http.NewRequest("DELETE", fmt.Sprintf("http://127.0.0.1:2001/healthcheck/%s", c), nil)
		if err != nil {
			t.Fatalf("Fail to build the HTTP request\n%v", err)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("HTTP request failed\n%v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("HTTP request failed, status %d", resp.StatusCode)
		}
		defer resp.Body.Close()
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("Fail to read the body\n%v", err)
		}
		body := string(bodyBytes)
		expected := fmt.Sprintf(`{"message":"Successfully deleted healthcheck %s"}`, c)
		if !strings.Contains(body, expected) {
			t.Fatalf("Invalid error message\n%s\n%s", expected, body)
		}
	}
	if len(healthcheck.Healthchecks) != 0 {
		t.Fatalf("Healthchecks were not successfully deleted: %d", len(healthcheck.Healthchecks))
	}
	err = component.Stop()
	if err != nil {
		t.Fatalf("Fail to stop the component\n%v", err)
	}
}

func TestOneOffCheck(t *testing.T) {
	count := 0
	logger := zap.NewExample()
	prom, err := prometheus.New()
	if err != nil {
		t.Fatalf("Error creating prometheus component :\n%v", err)
	}
	if err != nil {
		t.Fatalf("Error creating prometheus component :\n%v", err)
	}
	healthcheck, err := healthcheck.New(logger, make(chan *healthcheck.Result, 10), prom)
	if err != nil {
		t.Fatalf("Fail to create the healthcheck component\n%v", err)
	}
	component, err := New(zap.NewExample(), memorystore.NewMemoryStore(logger), prom, &Configuration{Host: "127.0.0.1", Port: 2001}, healthcheck)
	if err != nil {
		t.Fatalf("Fail to create the component\n%v", err)
	}
	err = component.Start()
	if err != nil {
		t.Fatalf("Fail to start the component\n%v", err)
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	port, err := strconv.ParseUint(strings.Split(ts.URL, ":")[2], 10, 16)
	if err != nil {
		t.Fatalf("error getting HTTP server port :\n%v", err)
	}
	client := &http.Client{}
	reqBody := fmt.Sprintf(`{"name":"baz","description":"bar","interval":"10m","one-off":true,"target":"127.0.0.1","port":%d,"timeout":"10s","protocol":"http","valid-status":[200]}`, port)
	req, err := http.NewRequest("POST", "http://127.0.0.1:2001/healthcheck/http", bytes.NewBuffer([]byte(reqBody)))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		t.Fatalf("Fail to build the HTTP request\n%v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("HTTP request failed\n%v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("HTTP request failed, status %d", resp.StatusCode)
	}
	if count != 1 {
		t.Fatalf("The target server was not reached: %d", count)
	}
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Fail to read the body\n%v", err)
	}
	body := string(bodyBytes)
	if !strings.Contains(body, "One-off healthcheck baz successfully executed") {
		t.Fatalf("Invalid body %s", body)
	}
	err = component.Stop()
	if err != nil {
		t.Fatalf("Fail to stop the component\n%v", err)
	}
}

func TestBulkEndpoint(t *testing.T) {
	logger := zap.NewExample()
	prom, err := prometheus.New()
	if err != nil {
		t.Fatalf("Error creating prometheus component :\n%v", err)
	}
	healthcheck, err := healthcheck.New(logger, make(chan *healthcheck.Result, 10), prom)
	if err != nil {
		t.Fatalf("Fail to create the healthcheck component\n%v", err)
	}
	component, err := New(zap.NewExample(), memorystore.NewMemoryStore(logger), prom, &Configuration{Host: "127.0.0.1", Port: 2001}, healthcheck)
	if err != nil {
		t.Fatalf("Fail to create the component\n%v", err)
	}
	err = component.Start()
	if err != nil {
		t.Fatalf("Fail to start the component\n%v", err)
	}

	client := &http.Client{}
	reqBody := `{"http-checks": [{"name":"baz","description":"bar","interval":"10m","target":"127.0.0.1","port":3000,"timeout":"10s","protocol":"http","valid-status":[200]}]}`
	req, err := http.NewRequest("POST", "http://127.0.0.1:2001/healthcheck/bulk", bytes.NewBuffer([]byte(reqBody)))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		t.Fatalf("Fail to build the HTTP request\n%v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("HTTP request failed\n%v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("HTTP request failed, status %d", resp.StatusCode)
	}
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Fail to read the body\n%v", err)
	}
	body := string(bodyBytes)
	if !strings.Contains(body, "Healthchecks successfully added") {
		t.Fatalf("Invalid body %s", body)
	}
	if len(healthcheck.Healthchecks) != 1 {
		t.Fatalf("Healthchecks were not successfully created: %d", len(healthcheck.Healthchecks))
	}
	err = component.Stop()
	if err != nil {
		t.Fatalf("Fail to stop the component\n%v", err)
	}
}

func basicAuth(username, password string) string {
	auth := username + ":" + password
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

func TestBasicAuth(t *testing.T) {
	prom, err := prometheus.New()
	if err != nil {
		t.Fatalf("Error creating prometheus component :\n%v", err)
	}
	logger := zap.NewExample()
	memstore := memorystore.NewMemoryStore(logger)
	healthcheck, err := healthcheck.New(zap.NewExample(), make(chan *healthcheck.Result, 10), prom)
	if err != nil {
		t.Fatalf("Fail to create the healthcheck component\n%v", err)
	}
	component, err := New(logger,
		memstore,
		prom,
		&Configuration{
			Host: "127.0.0.1",
			Port: 2001,
			BasicAuth: BasicAuth{
				Username: "foobar",
				Password: "mypassword",
			}},
		healthcheck)
	if err != nil {
		t.Fatalf("Fail to create the component\n%v", err)
	}
	err = component.Start()
	if err != nil {
		t.Fatalf("Fail to start the component\n%v", err)
	}
	resp, err := http.Get("http://127.0.0.1:2001/result")
	if err != nil {
		t.Fatalf("HTTP request failed\n%v", err)
	}
	if resp.StatusCode != 401 {
		t.Fatalf("Expected 401, got status %d", resp.StatusCode)
	}
	req, err := http.NewRequest("GET", "http://127.0.0.1:2001/result", nil)
	if err != nil {
		t.Fatalf("Fail to build the request\n%v", err)
	}
	req.Header.Add("Authorization", "Basic "+basicAuth("foobar", "mypassword"))
	client := &http.Client{}
	resp, err = client.Do(req)
	if err != nil {
		t.Fatalf("HTTP request failed\n%v", err)
	}
	if resp.StatusCode != 200 {
		t.Fatalf("Expected 200, got status %d", resp.StatusCode)
	}
}
