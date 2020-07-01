package healthcheck

import (
	"testing"
	"time"

	"go.uber.org/zap"

	"cabourotte/prometheus"
)

func TestStartStop(t *testing.T) {
	component, err := New(zap.NewExample(), make(chan *Result, 10), prometheus.New())
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

func TestAddRemoveCheck(t *testing.T) {
	logger := zap.NewExample()
	component, err := New(logger, make(chan *Result, 10), prometheus.New())
	if err != nil {
		t.Fatalf("Fail to create the component\n%v", err)
	}
	err = component.Start()
	if err != nil {
		t.Fatalf("Fail to start the component\n%v", err)
	}
	healthcheck := NewTCPHealthcheck(
		logger,
		&TCPHealthcheckConfiguration{
			Name:        "foo",
			Description: "bar",
			Target:      "127.0.0.1",
			Port:        9000,
			Timeout:     Duration(time.Second * 3),
			Interval:    Duration(time.Second * 5),
			OneOff:      false,
		},
	)
	err = component.AddCheck(healthcheck)
	if err != nil {
		t.Fatalf("Fail to add the healthcheck\n%v", err)
	}
	if len(component.Healthchecks) != 1 {
		t.Fatalf("The healthcheck was not added")
	}
	listResult := component.ListChecks()
	if len(listResult) != 1 {
		t.Fatalf("The healthcheck is not in the healthcheck list")
	}
	if listResult[0].Name() != "foo" {
		t.Fatalf("The healthcheck name is not accurate")
	}
	newHealthcheck := NewTCPHealthcheck(
		logger,
		&TCPHealthcheckConfiguration{
			Name:        "foo",
			Description: "bar",
			Target:      "127.0.0.1",
			Port:        9000,
			Timeout:     Duration(time.Second * 3),
			Interval:    Duration(time.Second * 5),
			OneOff:      false,
		},
	)
	// add replaces the existing healthcheck
	err = component.AddCheck(newHealthcheck)
	if err != nil {
		t.Fatalf("Fail to add the healthcheck\n%v", err)
	}
	if len(component.Healthchecks) != 1 {
		t.Fatalf("The healthcheck was not added")
	}
	// test removing the healthcheck
	err = component.RemoveCheck("foo")
	if err != nil {
		t.Fatalf("Fail to remove the healthcheck\n%v", err)
	}
	if len(component.Healthchecks) != 0 {
		t.Fatalf("The healthcheck was not removed")
	}
	// remove is idempotent
	err = component.RemoveCheck("foo")
	if err != nil {
		t.Fatalf("Fail to remove the healthcheck\n%v", err)
	}
	if len(component.Healthchecks) != 0 {
		t.Fatalf("The healthcheck was not removed")
	}
	err = component.Stop()
	if err != nil {
		t.Fatalf("Fail to stop the component\n%v", err)
	}
}
