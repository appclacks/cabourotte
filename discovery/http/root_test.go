package http

import (
	"encoding/json"
	"net"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/mcorbin/cabourotte/healthcheck"
	"github.com/mcorbin/cabourotte/prometheus"
)

func TestRequest(t *testing.T) {
	firstResultPayload := ResultPayload{
		DNSChecks: []healthcheck.DNSHealthcheckConfiguration{
			healthcheck.DNSHealthcheckConfiguration{
				Base: healthcheck.Base{
					Name:        "foo",
					Description: "bar",
					Interval:    healthcheck.Duration(time.Second * 10),
				},
				Timeout: healthcheck.Duration(time.Second * 2),
				Domain:  "mcorbin.fr",
			},
		},
	}
	secondResultPayload := ResultPayload{
		DNSChecks: []healthcheck.DNSHealthcheckConfiguration{
			healthcheck.DNSHealthcheckConfiguration{
				Base: healthcheck.Base{
					Name:        "new",
					Description: "bar",
					Interval:    healthcheck.Duration(time.Second * 10),
				},
				Timeout: healthcheck.Duration(time.Second * 2),
				Domain:  "mcorbin.fr",
			},
		},
		TCPChecks: []healthcheck.TCPHealthcheckConfiguration{
			healthcheck.TCPHealthcheckConfiguration{
				Base: healthcheck.Base{
					Name:        "tcp",
					Description: "bar",
					Interval:    healthcheck.Duration(time.Second * 10),
					Labels: map[string]string{
						"environment": "prod",
					},
				},
				Target:   "127.0.0.1",
				Port:     8080,
				SourceIP: healthcheck.IP(net.ParseIP("10.0.0.4")),
				Timeout:  healthcheck.Duration(time.Second * 5),
			},
		},
	}
	prom, err := prometheus.New()
	if err != nil {
		t.Fatalf("Error creating prometheus component :\n%v", err)
	}
	logger := zap.NewExample()
	checkComponent, err := healthcheck.New(logger, make(chan *healthcheck.Result, 10), prom)
	if err != nil {
		t.Fatalf("Fail to create the healthcheck component\n%v", err)
	}
	count := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			w.WriteHeader(http.StatusInternalServerError)
		} else {
			var payload ResultPayload
			if count == 0 {
				payload = firstResultPayload
			} else {
				payload = secondResultPayload
			}
			body, err := json.Marshal(payload)
			if err != nil {
				t.Fatalf("Error marshaling to json\n%v", err)
			}
			_, err = w.Write([]byte(body))
			if err != nil {
				t.Fatalf("Error writing body:\n%v", err)
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
	discoveryConfig := Configuration{
		Host:     "127.0.0.1",
		Path:     "/",
		Port:     uint32(port),
		Protocol: healthcheck.HTTP,
		Interval: 10,
	}
	discovery, err := New(logger, &discoveryConfig, checkComponent, prom)
	if err != nil {
		t.Fatalf("Fail to create the HTTP discovery component :\n%v", err)
	}
	err = discovery.request()
	if err != nil {
		t.Fatalf("HTTP discovery request failed\n%v", err)
	}
	checks := checkComponent.ListChecks()
	if len(checks) != 1 {
		t.Fatalf("Expected 1 configured healthchecks, got %d", len(checks))
	}
	if checks[0].Base().Name != "foo" {
		t.Fatalf("Invalid healthcheck name %s", checks[0].Base().Name)
	}
	err = discovery.request()
	if err != nil {
		t.Fatalf("HTTP discovery request failed\n%v", err)
	}
	checks = checkComponent.ListChecks()
	if len(checks) != 2 {
		t.Fatalf("Expected 2 configured healthchecks, got %d", len(checks))
	}
	if checks[0].Base().Name != "tcp" && checks[1].Base().Name != "tcp" {
		t.Fatalf("Invalid healthcheck names: %s, %s",
			checks[0].Base().Name,
			checks[1].Base().Name,
		)
	}
	if checks[0].Base().Name != "new" && checks[1].Base().Name != "new" {
		t.Fatalf("Invalid healthcheck names: %s, %s",
			checks[0].Base().Name,
			checks[1].Base().Name,
		)
	}
}
