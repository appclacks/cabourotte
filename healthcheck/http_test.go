package healthcheck

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestisSuccessfulOK(t *testing.T) {
	h := HTTPHealthcheck{
		config: &HTTPHealthcheckConfiguration{
			ValidStatus: []uint{200},
		},
	}
	response := http.Response{StatusCode: 200}
	if !h.isSuccessful(&response) {
		t.Errorf("Invalid status check")
	}

	h = HTTPHealthcheck{
		config: &HTTPHealthcheckConfiguration{
			ValidStatus: []uint{200, 201, 400},
		},
	}
	response = http.Response{StatusCode: 400}
	if !h.isSuccessful(&response) {
		t.Errorf("Invalid status check")
	}
}

func TestIssuccessfulFailure(t *testing.T) {
	h := HTTPHealthcheck{
		config: &HTTPHealthcheckConfiguration{
			ValidStatus: []uint{200},
		},
	}
	response := http.Response{StatusCode: 201}
	if h.isSuccessful(&response) {
		t.Errorf("Invalid status check")
	}

	h = HTTPHealthcheck{
		config: &HTTPHealthcheckConfiguration{
			ValidStatus: []uint{200, 201, 400},
		},
	}
	response = http.Response{StatusCode: 500}
	if h.isSuccessful(&response) {
		t.Errorf("Invalid status check")
	}
}

func TestHTTPExecuteSuccess(t *testing.T) {
	count := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	port, err := strconv.ParseUint(strings.Split(ts.URL, ":")[2], 10, 16)
	if err != nil {
		t.Errorf("error getting HTTP server port :\n%v", err)
	}
	h := HTTPHealthcheck{
		Logger: zap.NewExample(),
		config: &HTTPHealthcheckConfiguration{
			ValidStatus: []uint{200},
			Port:        uint(port),
			Target:      "127.0.0.1",
			Protocol:    HTTP,
			Path:        "/",
			Timeout:     time.Second * 2,
		},
	}
	h.buildURL()
	err = h.Execute()
	if err != nil {
		t.Errorf("healthcheck error :\n%v", err)
	}
	if count != 1 {
		t.Errorf("The request counter is invalid")
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
		t.Errorf("error getting HTTP server port :\n%v", err)
	}
	h := HTTPHealthcheck{
		Logger: zap.NewExample(),
		ID:     "bar",
		config: &HTTPHealthcheckConfiguration{
			Name:        "foo",
			ValidStatus: []uint{200},
			Port:        uint(port),
			Target:      "127.0.0.1",
			Protocol:    HTTP,
			Path:        "/",
			Timeout:     time.Second * 2,
		},
	}
	h.buildURL()
	err = h.Execute()
	if err == nil {
		t.Errorf("Was expecting an error :\n%v", err)
	}
	if count != 1 {
		t.Errorf("The request counter is invalid")
	}
}

func TestHTTPBuildURL(t *testing.T) {
	h := HTTPHealthcheck{
		config: &HTTPHealthcheckConfiguration{
			ValidStatus: []uint{200},
			Port:        2000,
			Target:      "127.0.0.1",
			Protocol:    HTTP,
			Path:        "/",
			Timeout:     time.Second * 2,
		},
	}
	h.buildURL()
	expectedURL := "http://127.0.0.1:2000/"
	if h.URL != expectedURL {
		t.Errorf("Invalid URL\nexpected: %s\nactual: %s", expectedURL, h.URL)
	}
}

func TestHTTPSBuildURL(t *testing.T) {
	h := HTTPHealthcheck{
		config: &HTTPHealthcheckConfiguration{
			ValidStatus: []uint{200},
			Port:        2000,
			Target:      "127.0.0.1",
			Protocol:    HTTPS,
			Path:        "/foo",
			Timeout:     time.Second * 2,
		},
	}
	h.buildURL()
	expectedURL := "https://127.0.0.1:2000/foo"
	if h.URL != expectedURL {
		t.Errorf("Invalid URL\nexpected: %s\nactual: %s", expectedURL, h.URL)
	}
}
