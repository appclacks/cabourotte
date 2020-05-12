package healthcheck

import (
	"net"
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
		Config: &TCPHealthcheckConfiguration{
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
		Config: &TCPHealthcheckConfiguration{
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
func TestTCPv6ExecuteSuccess(t *testing.T) {
	l, err := net.Listen("tcp", "[::1]:0")
	if err != nil {
		t.Error("fail to listen :\n/v", err)
	}
	ts := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	ts.Listener.Close()
	ts.Listener = l
	ts.Start()
	defer ts.Close()

	splitted := strings.Split(ts.URL, ":")
	port, err := strconv.ParseUint(splitted[len(splitted)-1], 10, 16)
	if err != nil {
		t.Errorf("error getting HTTP server port :\n%v", err)
	}
	h := TCPHealthcheck{
		Logger: zap.NewExample(),
		Config: &TCPHealthcheckConfiguration{
			Port:    uint(port),
			Target:  "::1",
			Timeout: time.Second * 2,
		},
	}
	h.buildURL()
	err = h.Execute()
	if err != nil {
		t.Errorf("healthcheck error :\n%v", err)
	}
}

func TestTCPStartStop(t *testing.T) {
	logger := zap.NewExample()
	healthcheck := NewTCPHealthcheck(
		logger,
		make(chan *Result, 10),
		&TCPHealthcheckConfiguration{
			Name:        "foo",
			Description: "bar",
			Target:      "127.0.0.1",
			Port:        9000,
			Timeout:     time.Second * 3,
			Interval:    time.Second * 5,
			OneOff:      false,
		},
	)
	err := healthcheck.Start()
	if err != nil {
		t.Errorf("Fail to start the healthcheck\n%v", err)
	}
	err = healthcheck.Stop()
	if err != nil {
		t.Errorf("Fail to stop the healthcheck\n%v", err)
	}
}
