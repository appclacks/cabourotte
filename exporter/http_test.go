package exporter

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"

	"cabourotte/healthcheck"
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
		t.Errorf("Error getting HTTP server port :\n%v", err)
	}
	exporter := NewHTTPExporter(
		zap.NewExample(),
		&HTTPConfiguration{
			Host:     "127.0.0.1",
			Port:     uint32(port),
			Protocol: healthcheck.HTTP,
		})
	err = exporter.Start()
	if err != nil {
		t.Errorf("Fail to start the http exporter:\n%v", err)
	}
	err = exporter.Push(&healthcheck.Result{
		Name:      "foo",
		Success:   true,
		Timestamp: time.Now(),
		Message:   "message",
	})
	if err != nil {
		t.Errorf("Fail to push healthcheck result:\n%v", err)
	}
	err = exporter.Stop()
	if err != nil {
		t.Errorf("Fail to stop the http exporter:\n%v", err)
	}
	if count != 1 {
		t.Errorf("The request counter is invalid")
	}
}
