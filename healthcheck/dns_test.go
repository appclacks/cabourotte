package healthcheck

import (
	"testing"
	"time"

	"go.uber.org/zap"

	"cabourotte/prometheus"
)

func TestDNSExecuteSuccess(t *testing.T) {
	h := DNSHealthcheck{
		Logger: zap.NewExample(),
		Config: &DNSHealthcheckConfiguration{
			// it will hopefully resolve ^^
			Domain: "mcorbin.fr",
		},
	}

	err := h.Execute()
	if err != nil {
		t.Fatalf("healthcheck error :\n%v", err)
	}
}

func TestDNSExecuteFailure(t *testing.T) {
	h := DNSHealthcheck{
		Logger: zap.NewExample(),
		Config: &DNSHealthcheckConfiguration{
			Domain: "doesnotexist.mcorbin.fr",
		},
	}

	err := h.Execute()
	if err == nil {
		t.Fatalf("Was expecting an error: the domain does not exist")
	}
}

func TestDNSStartStop(t *testing.T) {
	logger := zap.NewExample()
	healthcheck := NewDNSHealthcheck(
		logger,
		&DNSHealthcheckConfiguration{
			Name:        "foo",
			Description: "bar",
			Domain:      "mcorbin.fr",
			Interval:    Duration(time.Second * 5),
			OneOff:      false,
		},
	)
	wrapper := NewWrapper(healthcheck)
	component, err := New(zap.NewExample(), make(chan *Result, 10), prometheus.New())
	if err != nil {
		t.Fatalf("Fail to create the component\n%v", err)
	}
	component.startWrapper(wrapper)
	err = wrapper.Stop()
	if err != nil {
		t.Fatalf("Fail to stop the healthcheck\n%v", err)
	}
}
