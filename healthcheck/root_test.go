package healthcheck

import (
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/mcorbin/cabourotte/prometheus"
)

func TestStartStop(t *testing.T) {
	prom, err := prometheus.New()
	if err != nil {
		t.Fatalf("Error creating prometheus component :\n%v", err)
	}
	component, err := New(zap.NewExample(), make(chan *Result, 10), prom, []string{})
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
	prom, err := prometheus.New()
	if err != nil {
		t.Fatalf("Error creating prometheus component :\n%v", err)
	}
	component, err := New(logger, make(chan *Result, 10), prom, []string{})
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
	if listResult[0].Base().Name != "foo" {
		t.Fatalf("The healthcheck name is not accurate")
	}
	newHealthcheck := NewTCPHealthcheck(
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

func TestGetCheck(t *testing.T) {
	logger := zap.NewExample()
	prom, err := prometheus.New()
	if err != nil {
		t.Fatalf("Error creating prometheus component :\n%v", err)
	}
	component, err := New(logger, make(chan *Result, 10), prom, []string{})
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
	err = component.AddCheck(healthcheck)
	if err != nil {
		t.Fatalf("Fail to add the healthcheck\n%v", err)
	}
	if len(component.Healthchecks) != 1 {
		t.Fatalf("The healthcheck was not added")
	}
	check := component.GetCheck("foo")
	if check == nil {
		t.Fatalf("The healthcheck was not found")
	}
	check = component.GetCheck("notfound")
	if check != nil {
		t.Fatalf("The healthcheck should be missing, and so GetCheck returns an error")
	}
}

func TestMergeLabels(t *testing.T) {
	b := Base{
		Labels: nil,
	}
	MergeLabels(&b, map[string]string{})
	if b.Labels == nil {
		t.Fatalf("Merge failed")
	}
	if len(b.Labels) != 0 {
		t.Fatalf("Merge failed")
	}
	b = Base{
		Labels: map[string]string{},
	}
	MergeLabels(&b, nil)
	if len(b.Labels) != 0 {
		t.Fatalf("Merge failed")
	}
	b = Base{
		Labels: map[string]string{"a": "b"},
	}
	MergeLabels(&b, map[string]string{
		"foo": "bar",
	})
	if len(b.Labels) != 2 {
		t.Fatalf("Merge failed")
	}
	if b.Labels["a"] != "b" {
		t.Fatalf("Merge failed")
	}
	if b.Labels["foo"] != "bar" {
		t.Fatalf("Merge failed")
	}

}
