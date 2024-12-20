package healthcheck

import (
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestCommandExecuteSuccess(t *testing.T) {
	h := CommandHealthcheck{
		Logger: zap.NewExample(),
		Config: &CommandHealthcheckConfiguration{
			Command: "ls",
			Timeout: Duration(time.Second * 2),
		},
	}
	eErr := h.Execute()
	if eErr.Error != nil {
		t.Fatalf("healthcheck error :\n%v", eErr.Error)
	}
}

func TestCommandExecuteFailure(t *testing.T) {
	h := CommandHealthcheck{
		Logger: zap.NewExample(),
		Config: &CommandHealthcheckConfiguration{
			Command:   "ls",
			Arguments: []string{"/doesnotexist"},
			Timeout:   Duration(time.Second * 2),
		},
	}
	eErr := h.Execute()
	if eErr.Error == nil {
		t.Fatalf("healthcheck was expected to fail")
	}
}
