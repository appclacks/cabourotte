package healthcheck

import (
	"testing"

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
