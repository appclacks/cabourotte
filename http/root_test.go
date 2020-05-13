package http

import (
	"testing"

	"go.uber.org/zap"

	"cabourotte/healthcheck"
)

func TestStartStop(t *testing.T) {
	healthcheck, err := healthcheck.New(zap.NewExample(), make(chan *healthcheck.Result, 10))
	if err != nil {
		t.Errorf("Fail to create the healthcheck component\n%v", err)
	}
	component, err := New(zap.NewExample(), &Configuration{Host: "127.0.0.1", Port: 2000}, healthcheck)
	if err != nil {
		t.Errorf("Fail to create the component\n%v", err)
	}
	err = component.Start()
	if err != nil {
		t.Errorf("Fail to start the component\n%v", err)
	}
	err = component.Stop()
	if err != nil {
		t.Errorf("Fail to stop the component\n%v", err)
	}
}
