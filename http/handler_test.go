package http

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"go.uber.org/zap"

	"cabourotte/healthcheck"
)

func TestHandlers(t *testing.T) {
	healthcheck, err := healthcheck.New(zap.NewExample(), make(chan *healthcheck.Result, 10))
	if err != nil {
		t.Errorf("Fail to create the healthcheck component\n%v", err)
	}
	component, err := New(zap.NewExample(), &Configuration{Host: "127.0.0.1", Port: 2000}, healthcheck)
	if err != nil {
		t.Errorf("Fail to create the component\n%v", err)
	}
	err = component.Start()
	if err != nil {
		t.Errorf("Fail to start the component\n%v", err)
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
			payload:  `{"name":"bar","description":"bar","domain":"mcorbin.fr","interval":"10m","one-off":false,"target":"mcorbin.fr","port":9999,"timeout":"10s"}`,
		},
		{
			endpoint: "/healthcheck/http",
			payload:  `{"name":"baz","description":"bar","domain":"mcorbin.fr","interval":"10m","one-off":false,"target":"mcorbin.fr","port":9999,"timeout":"10s","protocol":"http","valid-status":[200]}`,
		},
	}
	client := &http.Client{}
	for _, c := range cases {
		req, err := http.NewRequest("POST", fmt.Sprintf("http://127.0.0.1:2000%s", c.endpoint), bytes.NewBuffer([]byte(c.payload)))
		req.Header.Set("Content-Type", "application/json")
		if err != nil {
			t.Errorf("Fail to build the HTTP request\n%v", err)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Errorf("HTTP request failed\n%v", err)
		}
		if resp.StatusCode != http.StatusCreated {
			t.Errorf("HTTP request failed, status %d", resp.StatusCode)
		}
	}
	if len(healthcheck.Healthchecks) != 3 {
		t.Errorf("Healthchecks were not successfully created: %d", len(healthcheck.Healthchecks))
	}

	// get the healthchecks
	resp, err := http.Get("http://127.0.0.1:2000/healthcheck")
	if err != nil {
		t.Errorf("Fail to get the healthchecks\n%v", err)
	}
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Fail to read the body\n%v", err)
	}
	body := string(bodyBytes)
	if !strings.Contains(body, `"name":"foo"`) {
		t.Errorf("Invalid body\n")
	}
	if !strings.Contains(body, `"name":"bar"`) {
		t.Errorf("Invalid body\n")
	}
	if !strings.Contains(body, `"name":"baz"`) {
		t.Errorf("Invalid body\n")
	}
	// delete everything
	checks := []string{"foo", "bar", "baz"}
	for _, c := range checks {
		req, err := http.NewRequest("DELETE", fmt.Sprintf("http://127.0.0.1:2000/healthcheck/%s", c), nil)
		if err != nil {
			t.Errorf("Fail to build the HTTP request\n%v", err)
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Errorf("HTTP request failed\n%v", err)
		}
		if resp.StatusCode != http.StatusOK {
			t.Errorf("HTTP request failed, status %d", resp.StatusCode)
		}
		defer resp.Body.Close()
		bodyBytes, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Errorf("Fail to read the body\n%v", err)
		}
		body := string(bodyBytes)
		expected := fmt.Sprintf(`{"message":"Successfully deleted healthcheck %s"}`, c)
		if !strings.Contains(body, expected) {
			t.Errorf("Invalid error message\n%s\n%s", expected, body)
		}
	}
	if len(healthcheck.Healthchecks) != 0 {
		t.Errorf("Healthchecks were not successfully deleted: %d", len(healthcheck.Healthchecks))
	}
	err = component.Stop()
	if err != nil {
		t.Errorf("Fail to stop the component\n%v", err)
	}
}

func TestOneOffCheck(t *testing.T) {
	count := 0
	healthcheck, err := healthcheck.New(zap.NewExample(), make(chan *healthcheck.Result, 10))
	if err != nil {
		t.Errorf("Fail to create the healthcheck component\n%v", err)
	}
	component, err := New(zap.NewExample(), &Configuration{Host: "127.0.0.1", Port: 2000}, healthcheck)
	if err != nil {
		t.Errorf("Fail to create the component\n%v", err)
	}
	err = component.Start()
	if err != nil {
		t.Errorf("Fail to start the component\n%v", err)
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	port, err := strconv.ParseUint(strings.Split(ts.URL, ":")[2], 10, 16)
	if err != nil {
		t.Errorf("error getting HTTP server port :\n%v", err)
	}
	client := &http.Client{}
	reqBody := fmt.Sprintf(`{"name":"baz","description":"bar","domain":"mcorbin.fr","interval":"10m","one-off":true,"target":"mcorbin.fr","port":%d,"timeout":"10s","protocol":"http","valid-status":[200]}`, port)
	req, err := http.NewRequest("POST", "http://127.0.0.1:2000/healthcheck/http", bytes.NewBuffer([]byte(reqBody)))
	req.Header.Set("Content-Type", "application/json")
	if err != nil {
		t.Errorf("Fail to build the HTTP request\n%v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Errorf("HTTP request failed\n%v", err)
	}
	if resp.StatusCode != http.StatusCreated {
		t.Errorf("HTTP request failed, status %d", resp.StatusCode)
	}
	if count != 1 {
		t.Errorf("The target server was not reached: %d", count)
	}
	defer resp.Body.Close()
	bodyBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Errorf("Fail to read the body\n%v", err)
	}
	body := string(bodyBytes)
	if !strings.Contains(body, "One-off healthcheck baz successfully executed") {
		t.Errorf("Invalid body %s", body)
	}
}