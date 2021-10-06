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
	err := h.Execute()
	if err != nil {
		t.Fatalf("healthcheck error :\n%v", err)
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
	err := h.Execute()
	if err == nil {
		t.Fatalf("healthcheck was expected to fail")
	}
}
