package healthcheck

import (
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestCommandExecuteSuccess(t *testing.T) {
	h := CommandHealthcheck{
		Base{
			Logger: zap.NewExample(),
			Config: &CommandHealthcheckConfiguration{
				BaseConfig: BaseConfig{
					Timeout: Duration(time.Second * 2),
				},
				Command: "ls",
			},
		},
	}
	err := h.Execute()
	if err != nil {
		t.Fatalf("healthcheck error :\n%v", err)
	}
}

func TestCommandExecuteFailure(t *testing.T) {
	h := CommandHealthcheck{
		Base{
			Logger: zap.NewExample(),
			Config: &CommandHealthcheckConfiguration{
				Command:   "ls",
				Arguments: []string{"/doesnotexist"},
				BaseConfig: BaseConfig{
					Timeout: Duration(time.Second * 2),
				},
			},
		},
	}
	err := h.Execute()
	if err == nil {
		t.Fatalf("healthcheck was expected to fail")
	}
}
