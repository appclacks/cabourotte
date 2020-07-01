package http

import (
	"testing"

	"go.uber.org/zap"

	"cabourotte/healthcheck"
	"cabourotte/memorystore"
	"cabourotte/prometheus"
)

func TestStartStop(t *testing.T) {
	prom := prometheus.New()
	logger := zap.NewExample()
	healthcheck, err := healthcheck.New(logger, make(chan *healthcheck.Result, 10), prom)
	if err != nil {
		t.Fatalf("Fail to create the healthcheck component\n%v", err)
	}
	component, err := New(logger, memorystore.NewMemoryStore(logger), prom, &Configuration{Host: "127.0.0.1", Port: 2000}, healthcheck)
	if err != nil {
		t.Fatalf("Fail to create the component\n%v", err)
	}
	err = component.Start()
	if err != nil {
		t.Fatalf("Fail to start the component\n%v", err)
	}
	err = component.Stop()
	if err != nil {
		t.Fatalf("Fail to stop the component\n%v", err)
	}
}

func TestStartStopTLS(t *testing.T) {
	logger := zap.NewExample()
	prom := prometheus.New()
	healthcheck, err := healthcheck.New(logger, make(chan *healthcheck.Result, 10), prom)
	if err != nil {
		t.Fatalf("Fail to create the healthcheck component\n%v", err)
	}
	component, err := New(logger, memorystore.NewMemoryStore(logger), prom, &Configuration{Host: "127.0.0.1", Port: 2000, Key: "../test/key.pem", Cert: "../test/cert.pem", Cacert: "../test/cert.pem"}, healthcheck)
	if err != nil {
		t.Fatalf("Fail to create the component\n%v", err)
	}
	err = component.Start()
	if err != nil {
		t.Fatalf("Fail to start the component\n%v", err)
	}
	err = component.Stop()
	if err != nil {
		t.Fatalf("Fail to stop the component\n%v", err)
	}
}
