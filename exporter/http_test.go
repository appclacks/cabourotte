package exporter

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/appclacks/cabourotte/healthcheck"
)

func TestHTTPExporter(t *testing.T) {
	count := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count++
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	port, err := strconv.ParseUint(strings.Split(ts.URL, ":")[2], 10, 16)
	if err != nil {
		t.Fatalf("Error getting HTTP server port :\n%v", err)
	}
	exporter, err := NewHTTPExporter(
		zap.NewExample(),
		&HTTPConfiguration{
			Host:     "127.0.0.1",
			Port:     uint32(port),
			Protocol: healthcheck.HTTP,
		})
	if err != nil {
		t.Fatalf("Error creating the http exporter :\n%v", err)
	}
	err = exporter.Start()
	if err != nil {
		t.Fatalf("Fail to start the http exporter:\n%v", err)
	}
	err = exporter.Push(context.Background(), &healthcheck.Result{
		Name:                 "foo",
		Success:              true,
		HealthcheckTimestamp: time.Now().Unix(),
		Message:              "message",
	})
	if err != nil {
		t.Fatalf("Fail to push healthcheck result:\n%v", err)
	}
	err = exporter.Stop()
	if err != nil {
		t.Fatalf("Fail to stop the http exporter:\n%v", err)
	}
	if count != 1 {
		t.Fatalf("The request counter is invalid")
	}
}
