package healthcheck

import (
	"context"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/appclacks/cabourotte/prometheus"
)

func TestIsSuccessfulOK(t *testing.T) {
	h := HTTPHealthcheck{
		Config: &HTTPHealthcheckConfiguration{
			ValidStatus: []uint{200},
		},
	}
	response := http.Response{StatusCode: 200}
	if !h.isSuccessful(&response) {
		t.Fatalf("Invalid status check")
	}

	h = HTTPHealthcheck{
		Config: &HTTPHealthcheckConfiguration{
			ValidStatus: []uint{200, 201, 400},
		},
	}
	response = http.Response{StatusCode: 400}
	if !h.isSuccessful(&response) {
		t.Fatalf("Invalid status check")
	}
}

func TestIssuccessfulFailure(t *testing.T) {
	h := HTTPHealthcheck{
		Config: &HTTPHealthcheckConfiguration{
			ValidStatus: []uint{200},
		},
	}
	response := http.Response{StatusCode: 201}
	if h.isSuccessful(&response) {
		t.Fatalf("Invalid status check")
	}

	h = HTTPHealthcheck{
		Config: &HTTPHealthcheckConfiguration{
			ValidStatus: []uint{200, 201, 400},
		},
	}
	response = http.Response{StatusCode: 500}
	if h.isSuccessful(&response) {
		t.Fatalf("Invalid status check")
	}
}

func TestHTTPExecuteGetSuccess(t *testing.T) {
	count := 0
	headersOK := false
	bodyOK := false
	expectedBody := "my custom body"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			if r.Header.Get("Foo") == "Bar" && r.Header.Get("User-agent") == "Cabourotte" {
				headersOK = true
			}
			bodyBytes, _ := io.ReadAll(r.Body)
			body := string(bodyBytes)
			if body == expectedBody {
				bodyOK = true
			}
			count++
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer ts.Close()

	port, err := strconv.ParseUint(strings.Split(ts.URL, ":")[2], 10, 16)
	if err != nil {
		t.Fatalf("error getting HTTP server port :\n%v", err)
	}
	h := HTTPHealthcheck{
		Logger: zap.NewExample(),
		Config: &HTTPHealthcheckConfiguration{
			ValidStatus: []uint{200},
			Headers:     map[string]string{"Foo": "Bar"},
			Port:        uint(port),
			Target:      "127.0.0.1",
			Protocol:    HTTP,
			Body:        expectedBody,
			Path:        "/",
			Timeout:     Duration(time.Second * 2),
		},
	}
	err = h.Initialize()
	if err != nil {
		t.Fatalf("Initialization error :\n%v", err)
	}
	ctx := context.Background()
	err = h.Execute(&ctx)
	if err != nil {
		t.Fatalf("healthcheck error :\n%v", err)
	}
	if count != 1 {
		t.Fatal("The request counter is invalid")
	}
	if !headersOK {
		t.Fatal("Invalid headers")
	}
	if !bodyOK {
		t.Fatal("Invalid body")
	}
}

func TestHTTPExecuteRegexpSuccess(t *testing.T) {
	count := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("github.com/appclacks/cabourotte/ !"))
		if err != nil {
			t.Fatalf("Error writing :\n%v", err)
		}
	}))
	defer ts.Close()

	port, err := strconv.ParseUint(strings.Split(ts.URL, ":")[2], 10, 16)
	if err != nil {
		t.Fatalf("error getting HTTP server port :\n%v", err)
	}
	r := regexp.MustCompile("github.com/appclacks/cabourotte/*")
	regexp := Regexp(*r)
	h := HTTPHealthcheck{
		Logger: zap.NewExample(),
		Config: &HTTPHealthcheckConfiguration{
			ValidStatus: []uint{200},
			Headers:     map[string]string{"Foo": "Bar"},
			Port:        uint(port),
			Target:      "127.0.0.1",
			BodyRegexp:  []Regexp{regexp},
			Protocol:    HTTP,
			Path:        "/",
			Timeout:     Duration(time.Second * 2),
		},
	}
	err = h.Initialize()
	if err != nil {
		t.Fatalf("Initialization error :\n%v", err)
	}
	ctx := context.Background()
	err = h.Execute(&ctx)
	if err != nil {
		t.Fatalf("healthcheck error :\n%v", err)
	}
	if count != 1 {
		t.Fatal("The request counter is invalid")
	}
}

func TestHTTPExecuteRegexpFailure(t *testing.T) {
	count := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("github.com/appclacks/cabourotte/ !"))
		if err != nil {
			t.Fatalf("Error writing :\n%v", err)
		}
	}))
	defer ts.Close()

	port, err := strconv.ParseUint(strings.Split(ts.URL, ":")[2], 10, 16)
	if err != nil {
		t.Fatalf("error getting HTTP server port :\n%v", err)
	}
	r := regexp.MustCompile("trololo*")
	regexp := Regexp(*r)
	h := HTTPHealthcheck{
		Logger: zap.NewExample(),
		Config: &HTTPHealthcheckConfiguration{
			ValidStatus: []uint{200},
			Headers:     map[string]string{"Foo": "Bar"},
			Port:        uint(port),
			Target:      "127.0.0.1",
			BodyRegexp:  []Regexp{regexp},
			Protocol:    HTTP,
			Path:        "/",
			Timeout:     Duration(time.Second * 2),
		},
	}
	err = h.Initialize()
	if err != nil {
		t.Fatalf("Initialization error :\n%v", err)
	}
	ctx := context.Background()
	err = h.Execute(&ctx)
	if err == nil {
		t.Fatalf("Was expecting an error")
	}

}

func TestHTTPv6ExecuteSuccess(t *testing.T) {
	count := 0
	l, err := net.Listen("tcp", "[::1]:0")
	if err != nil {
		t.Error("fail to listen :\n/v", err)
	}
	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		w.WriteHeader(http.StatusOK)
	}))
	ts.Listener.Close()
	ts.Listener = l
	ts.Start()
	defer ts.Close()

	splitted := strings.Split(ts.URL, ":")
	port, err := strconv.ParseUint(splitted[len(splitted)-1], 10, 16)
	if err != nil {
		t.Fatalf("error getting HTTP server port :\n%v", err)
	}
	h := HTTPHealthcheck{
		Logger: zap.NewExample(),
		Config: &HTTPHealthcheckConfiguration{
			ValidStatus: []uint{200},
			Port:        uint(port),
			Target:      "::1",
			Protocol:    HTTP,
			Path:        "/",
			Timeout:     Duration(time.Second * 2),
		},
	}
	err = h.Initialize()
	if err != nil {
		t.Fatalf("Initialization error :\n%v", err)
	}
	ctx := context.Background()
	err = h.Execute(&ctx)
	if err != nil {
		t.Fatalf("healthcheck error :\n%v", err)
	}
	if count != 1 {
		t.Fatalf("The request counter is invalid")
	}
}

func TestHTTPExecuteFailure(t *testing.T) {
	count := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer ts.Close()

	port, err := strconv.ParseUint(strings.Split(ts.URL, ":")[2], 10, 16)
	if err != nil {
		t.Fatalf("error getting HTTP server port :\n%v", err)
	}
	h := HTTPHealthcheck{
		Logger: zap.NewExample(),
		Config: &HTTPHealthcheckConfiguration{
			Base: Base{
				Name: "foo",
			},
			ValidStatus: []uint{200},
			Port:        uint(port),
			Target:      "127.0.0.1",
			Protocol:    HTTP,
			Path:        "/",
			Timeout:     Duration(time.Second * 2),
		},
	}
	err = h.Initialize()
	if err != nil {
		t.Fatalf("Initialization error :\n%v", err)
	}
	ctx := context.Background()
	err = h.Execute(&ctx)
	if err == nil {
		t.Fatalf("Was expecting an error")
	}
	if count != 1 {
		t.Fatalf("The request counter is invalid")
	}
}

func TestHTTPBuildURL(t *testing.T) {
	h := HTTPHealthcheck{
		Config: &HTTPHealthcheckConfiguration{
			ValidStatus: []uint{200},
			Port:        2000,
			Target:      "127.0.0.1",
			Protocol:    HTTP,
			Path:        "/",
			Timeout:     Duration(time.Second * 2),
		},
	}
	h.buildURL()
	expectedURL := "http://127.0.0.1:2000/"
	if h.URL != expectedURL {
		t.Fatalf("Invalid URL\nexpected: %s\nactual: %s", expectedURL, h.URL)
	}
}

func TestHTTPSBuildURL(t *testing.T) {
	h := HTTPHealthcheck{
		Config: &HTTPHealthcheckConfiguration{
			ValidStatus: []uint{200},
			Port:        2000,
			Target:      "127.0.0.1",
			Protocol:    HTTPS,
			Path:        "/foo",
			Timeout:     Duration(time.Second * 2),
		},
	}
	h.buildURL()
	expectedURL := "https://127.0.0.1:2000/foo"
	if h.URL != expectedURL {
		t.Fatalf("Invalid URL\nexpected: %s\nactual: %s", expectedURL, h.URL)
	}
}

func TestHTTPStartStop(t *testing.T) {
	logger := zap.NewExample()
	healthcheck := NewHTTPHealthcheck(
		logger,
		&HTTPHealthcheckConfiguration{
			Base: Base{
				Name:        "foo",
				Description: "bar",
				Interval:    Duration(time.Second * 5),
				OneOff:      false,
			},
			Target:   "127.0.0.1",
			Path:     "/",
			Protocol: HTTP,
			Port:     9000,
			Timeout:  Duration(time.Second * 3),
		},
	)
	err := healthcheck.Initialize()
	if err != nil {
		t.Fatalf("Initialization error :\n%v", err)
	}
	wrapper := NewWrapper(healthcheck)
	prom, err := prometheus.New()
	if err != nil {
		t.Fatalf("Error creating prometheus component :\n%v", err)
	}
	component, err := New(zap.NewExample(), make(chan *Result, 10), prom, []string{})
	if err != nil {
		t.Fatalf("Fail to create the component\n%v", err)
	}
	component.startWrapper(wrapper)
	err = wrapper.Stop()
	if err != nil {
		t.Fatalf("Fail to stop the healthcheck\n%v", err)
	}
}

func TestHTTPExecuteSourceIP(t *testing.T) {
	count := 0
	headersOK := false
	bodyOK := false
	sourceIPOK := false
	expectedBody := "my custom body"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Foo") == "Bar" && r.Header.Get("User-agent") == "Cabourotte" {
			headersOK = true
		}
		bodyBytes, _ := io.ReadAll(r.Body)
		body := string(bodyBytes)
		if body == expectedBody {
			bodyOK = true
		}
		if strings.Split(r.RemoteAddr, ":")[0] == "127.0.0.1" {
			sourceIPOK = true
		}
		count++
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	port, err := strconv.ParseUint(strings.Split(ts.URL, ":")[2], 10, 16)
	if err != nil {
		t.Fatalf("error getting HTTP server port :\n%v", err)
	}
	h := HTTPHealthcheck{
		Logger: zap.NewExample(),
		Config: &HTTPHealthcheckConfiguration{
			SourceIP:    IP(net.ParseIP("127.0.0.1")),
			ValidStatus: []uint{200},
			Headers:     map[string]string{"Foo": "Bar"},
			Port:        uint(port),
			Target:      "127.0.0.1",
			Protocol:    HTTP,
			Body:        expectedBody,
			Path:        "/",
			Timeout:     Duration(time.Second * 2),
		},
	}
	err = h.Initialize()
	if err != nil {
		t.Fatalf("Initialization error :\n%v", err)
	}
	ctx := context.Background()
	err = h.Execute(&ctx)
	if err != nil {
		t.Fatalf("healthcheck error :\n%v", err)
	}
	if count != 1 {
		t.Fatal("The request counter is invalid")
	}
	if !headersOK {
		t.Fatal("Invalid headers")
	}
	if !bodyOK {
		t.Fatal("Invalid body")
	}
	if !sourceIPOK {
		t.Fatalf("Invalid source IP")
	}
}

func TestHTTPExecutePostSuccess(t *testing.T) {
	count := 0
	headersOK := false
	bodyOK := false
	expectedBody := "my custom body"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			if r.Header.Get("Foo") == "Bar" && r.Header.Get("User-agent") == "Cabourotte" {
				headersOK = true
			}
			bodyBytes, _ := io.ReadAll(r.Body)
			body := string(bodyBytes)
			if body == expectedBody {
				bodyOK = true
			}
			count++
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer ts.Close()

	port, err := strconv.ParseUint(strings.Split(ts.URL, ":")[2], 10, 16)
	if err != nil {
		t.Fatalf("error getting HTTP server port :\n%v", err)
	}
	h := HTTPHealthcheck{
		Logger: zap.NewExample(),
		Config: &HTTPHealthcheckConfiguration{
			ValidStatus: []uint{200},
			Headers:     map[string]string{"Foo": "Bar"},
			Port:        uint(port),
			Target:      "127.0.0.1",
			Method:      "POST",
			Protocol:    HTTP,
			Body:        expectedBody,
			Path:        "/",
			Timeout:     Duration(time.Second * 2),
		},
	}
	err = h.Initialize()
	if err != nil {
		t.Fatalf("Initialization error :\n%v", err)
	}
	ctx := context.Background()
	err = h.Execute(&ctx)
	if err != nil {
		t.Fatalf("healthcheck error :\n%v", err)
	}
	if count != 1 {
		t.Fatal("The request counter is invalid")
	}
	if !headersOK {
		t.Fatal("Invalid headers")
	}
	if !bodyOK {
		t.Fatal("Invalid body")
	}
}

func TestHTTPExecuteQueryParam(t *testing.T) {
	count := 0
	headersOK := false
	bodyOK := false
	expectedBody := "my custom body"
	query := map[string]string{
		"a":       "b",
		"trololo": "huhu",
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		q := r.URL.Query()
		for k, v := range query {
			params, ok := q[k]
			if !ok {
				t.Fatalf("Query param %s not found", k)
			}
			if params[0] != v {
				t.Fatalf("Incorrect query param %s", k)
			}
		}
		if r.Method != "POST" {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			if r.Header.Get("Foo") == "Bar" && r.Header.Get("User-agent") == "Cabourotte" {
				headersOK = true
			}
			bodyBytes, _ := io.ReadAll(r.Body)
			body := string(bodyBytes)
			if body == expectedBody {
				bodyOK = true
			}
			count++
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer ts.Close()

	port, err := strconv.ParseUint(strings.Split(ts.URL, ":")[2], 10, 16)
	if err != nil {
		t.Fatalf("error getting HTTP server port :\n%v", err)
	}
	h := HTTPHealthcheck{
		Logger: zap.NewExample(),
		Config: &HTTPHealthcheckConfiguration{
			ValidStatus: []uint{200},
			Headers:     map[string]string{"Foo": "Bar"},
			Port:        uint(port),
			Target:      "127.0.0.1",
			Query:       query,
			Method:      "POST",
			Protocol:    HTTP,
			Body:        expectedBody,
			Path:        "/",
			Timeout:     Duration(time.Second * 2),
		},
	}
	err = h.Initialize()
	if err != nil {
		t.Fatalf("Initialization error :\n%v", err)
	}
	ctx := context.Background()
	err = h.Execute(&ctx)
	if err != nil {
		t.Fatalf("healthcheck error :\n%v", err)
	}
	if count != 1 {
		t.Fatal("The request counter is invalid")
	}
	if !headersOK {
		t.Fatal("Invalid headers")
	}
	if !bodyOK {
		t.Fatal("Invalid body")
	}
}
