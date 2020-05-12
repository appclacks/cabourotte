package healthcheck

import (
	"testing"
	"time"

	"go.uber.org/zap"
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
		t.Errorf("healthcheck error :\n%v", err)
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
		t.Errorf("Was expecting an error: the domain does not exist")
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
			Interval:    time.Second * 5,
			OneOff:      false,
		},
	)
	err := healthcheck.Start(make(chan *Result, 10))
	if err != nil {
		t.Errorf("Fail to start the healthcheck\n%v", err)
	}
	err = healthcheck.Stop()
	if err != nil {
		t.Errorf("Fail to stop the healthcheck\n%v", err)
	}
}
