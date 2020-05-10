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

func TestTCPBuildURL(t *testing.T) {
	h := TCPHealthcheck{
		config: &TCPHealthcheckConfiguration{
			Port:   2000,
			Target: "127.0.0.1",
		},
	}
	h.buildURL()
	expectedURL := "127.0.0.1:2000"
	if h.URL != expectedURL {
		t.Errorf("Invalid URL\nexpected: %s\nactual: %s", expectedURL, h.URL)
	}
}

func TestTCPExecuteSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	port, err := strconv.ParseUint(strings.Split(ts.URL, ":")[2], 10, 16)
	if err != nil {
		t.Errorf("error getting HTTP server port :\n%v", err)
	}
	h := TCPHealthcheck{
		Logger: zap.NewExample(),
		config: &TCPHealthcheckConfiguration{
			Port:    uint(port),
			Target:  "127.0.0.1",
			Timeout: time.Second * 2,
		},
	}
	h.buildURL()
	err = h.Execute()
	if err != nil {
		t.Errorf("healthcheck error :\n%v", err)
	}
}
