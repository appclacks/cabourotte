package healthcheck

import (
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestStartStop(t *testing.T) {
	component, err := New(zap.NewExample())
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

func TestAddRemoveCheck(t *testing.T) {
	logger := zap.NewExample()
	component, err := New(logger)
	if err != nil {
		t.Errorf("Fail to create the component\n%v", err)
	}
	err = component.Start()
	if err != nil {
		t.Errorf("Fail to start the component\n%v", err)
	}
	healthcheck := NewTCPHealthcheck(
		logger,
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
	err = component.AddCheck(&healthcheck)
	if err != nil {
		t.Errorf("Fail to add the healthcheck\n%v", err)
	}
	if len(component.Healthchecks) != 1 {
		t.Errorf("The healthcheck was not added")
	}
	newHealthcheck := NewTCPHealthcheck(
		logger,
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
	// add replaces the existing healthcheck
	err = component.AddCheck(&newHealthcheck)
	if err != nil {
		t.Errorf("Fail to add the healthcheck\n%v", err)
	}
	if len(component.Healthchecks) != 1 {
		t.Errorf("The healthcheck was not added")
	}
	// test removing the healthcheck
	err = component.RemoveCheck("foo")
	if err != nil {
		t.Errorf("Fail to remove the healthcheck\n%v", err)
	}
	if len(component.Healthchecks) != 0 {
		t.Errorf("The healthcheck was not removed")
	}
	// remove is idempotent
	err = component.RemoveCheck("foo")
	if err != nil {
		t.Errorf("Fail to remove the healthcheck\n%v", err)
	}
	if len(component.Healthchecks) != 0 {
		t.Errorf("The healthcheck was not removed")
	}
	err = component.Stop()
	if err != nil {
		t.Errorf("Fail to stop the component\n%v", err)
	}
}
