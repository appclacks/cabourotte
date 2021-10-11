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

func TestTLSBuildURL(t *testing.T) {
	h := TLSHealthcheck{
		Base: Base{
			Config: &TLSHealthcheckConfiguration{
				Port:   2000,
				Target: "127.0.0.1",
			},
		},
	}
	h.buildURL()
	expectedURL := "127.0.0.1:2000"
	if h.URL != expectedURL {
		t.Fatalf("Invalid URL\nexpected: %s\nactual: %s", expectedURL, h.URL)
	}
}

func TestTLSExecuteError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	port, err := strconv.ParseUint(strings.Split(ts.URL, ":")[2], 10, 16)
	if err != nil {
		t.Fatalf("error getting HTTP server port :\n%v", err)
	}
	h := TLSHealthcheck{
		Base: Base{
			Logger: zap.NewExample(),
			Config: &TLSHealthcheckConfiguration{
				Port:   uint(port),
				Target: "127.0.0.1",
				BaseConfig: BaseConfig{
					Timeout: Duration(time.Second * 2),
				},
			},
		},
	}
	h.buildURL()
	err = h.Execute()
	if err == nil {
		t.Fatalf("Was expecting an error")
	}
}

func TestTLSExecuteErrorNoTarget(t *testing.T) {
	h := TLSHealthcheck{
		Base: Base{
			Logger: zap.NewExample(),
			Config: &TLSHealthcheckConfiguration{
				Port:   80,
				Target: "doesnotexist.mcorbin.fr",
				BaseConfig: BaseConfig{
					Timeout: Duration(time.Second * 2),
				},
			},
		},
	}
	h.buildURL()
	err := h.Execute()
	if err == nil {
		t.Fatalf("Was expecting an error")
	}
}
