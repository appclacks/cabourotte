package healthcheck

import (
	"context"
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
	ctx := context.Background()
	err := h.Execute(&ctx)
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
	ctx := context.Background()
	err := h.Execute(&ctx)
	if err == nil {
		t.Fatalf("healthcheck was expected to fail")
	}
}
