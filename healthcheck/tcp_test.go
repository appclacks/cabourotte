package healthcheck

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/appclacks/cabourotte/prometheus"
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
		t.Fatalf("Invalid URL\nexpected: %s\nactual: %s", expectedURL, h.URL)
	}
}

func TestTCPExecuteSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	port, err := strconv.ParseUint(strings.Split(ts.URL, ":")[2], 10, 16)
	if err != nil {
		t.Fatalf("error getting HTTP server port :\n%v", err)
	}
	h := TCPHealthcheck{
		Logger: zap.NewExample(),
		Config: &TCPHealthcheckConfiguration{
			Port:    uint(port),
			Target:  "127.0.0.1",
			Timeout: Duration(time.Second * 2),
		},
	}
	h.buildURL()
	ctx := context.Background()
	err = h.Execute(&ctx)
	if err != nil {
		t.Fatalf("healthcheck error :\n%v", err)
	}
}

func TestTCPExecuteSuccessSourceIP(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	port, err := strconv.ParseUint(strings.Split(ts.URL, ":")[2], 10, 16)
	if err != nil {
		t.Fatalf("error getting HTTP server port :\n%v", err)
	}
	h := TCPHealthcheck{
		Logger: zap.NewExample(),
		Config: &TCPHealthcheckConfiguration{
			Port:     uint(port),
			SourceIP: IP(net.ParseIP("127.0.0.1")),
			Target:   "127.0.0.1",
			Timeout:  Duration(time.Second * 2),
		},
	}
	h.buildURL()
	ctx := context.Background()
	err = h.Execute(&ctx)
	if err != nil {
		t.Fatalf("healthcheck error :\n%v", err)
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
		t.Fatalf("error getting HTTP server port :\n%v", err)
	}
	h := TCPHealthcheck{
		Logger: zap.NewExample(),
		Config: &TCPHealthcheckConfiguration{
			Port:    uint(port),
			Target:  "::1",
			Timeout: Duration(time.Second * 2),
		},
	}
	h.buildURL()
	ctx := context.Background()
	err = h.Execute(&ctx)
	if err != nil {
		t.Fatalf("healthcheck error :\n%v", err)
	}
}

func TestTCPStartStop(t *testing.T) {
	logger := zap.NewExample()
	healthcheck := NewTCPHealthcheck(
		logger,
		&TCPHealthcheckConfiguration{
			Base: Base{
				Name:        "foo",
				Description: "bar",
				Interval:    Duration(time.Second * 5),
				OneOff:      false,
			},
			Target:  "127.0.0.1",
			Port:    9000,
			Timeout: Duration(time.Second * 3),
		},
	)
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

func TestTCPExecuteSuccessShoulddFail(t *testing.T) {
	h := TCPHealthcheck{
		Logger: zap.NewExample(),
		Config: &TCPHealthcheckConfiguration{
			ShouldFail: true,
			Port:       80,
			Target:     "doesnotexist.mcorbin.fr",
			Timeout:    Duration(time.Second * 2),
		},
	}
	h.buildURL()
	ctx := context.Background()
	err := h.Execute(&ctx)
	if err != nil {
		t.Fatalf("healthcheck error :\n%v", err)
	}
}
