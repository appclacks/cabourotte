package healthcheck

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// CommandHealthcheckConfiguration defines a COMMAND healthcheck configuration
type CommandHealthcheckConfiguration struct {
	BaseConfig `json:",inline"`
	Command    string   `json:"command"`
	Arguments  []string `json:"arguments"`
}

// CommandHealthcheck defines an HTTP healthcheck
type CommandHealthcheck struct {
	Base
}

// Validate validates the healthcheck configuration
func (config *CommandHealthcheckConfiguration) Validate() error {
	if err := config.BaseConfig.Validate(); err != nil {
		return err
	}
	if config.Command == "" {
		return errors.New("The healthcheck command is missing")
	}
	return nil
}

// Initialize the healthcheck.
func (h *CommandHealthcheck) Initialize() error {
	return nil
}

// Summary returns an healthcheck summary
func (h *CommandHealthcheck) Summary() string {
	summary := ""
	if h.Base.Config.GetDescription() != "" {
		summary = fmt.Sprintf("%s, command %s", h.Base.Config.GetDescription(), h.Base.Config.(*CommandHealthcheckConfiguration).Command)

	} else {
		summary = fmt.Sprintf("command %s", h.Base.Config.(*CommandHealthcheckConfiguration).Command)
	}

	return summary
}

// LogError logs an error with context
func (h *CommandHealthcheck) LogError(err error, message string) {
	h.Base.Logger.Error(err.Error(),
		zap.String("extra", message),
		zap.String("command", h.Base.Config.(*CommandHealthcheckConfiguration).Command),
		zap.String("name", h.Base.Config.GetName()))
}

// LogDebug logs a message with context
func (h *CommandHealthcheck) LogDebug(message string) {
	h.Base.Logger.Debug(message,
		zap.String("command", h.Base.Config.(*CommandHealthcheckConfiguration).Command),
		zap.String("name", h.Base.Config.GetName()))
}

// LogInfo logs a message with context
func (h *CommandHealthcheck) LogInfo(message string) {
	h.Base.Logger.Info(message,
		zap.String("command", h.Base.Config.(*CommandHealthcheckConfiguration).Command),
		zap.String("name", h.Base.Config.GetName()))
}

// Execute executes an healthcheck on the given domain
func (h *CommandHealthcheck) Execute() error {
	h.LogDebug("start executing healthcheck")
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(h.Base.Config.GetTimeout())*time.Second)
	defer cancel()
	var stdErr bytes.Buffer
	cmd := exec.CommandContext(ctx, h.Base.Config.(*CommandHealthcheckConfiguration).Command, h.Base.Config.(*CommandHealthcheckConfiguration).Arguments...)
	cmd.Stderr = &stdErr
	if err := cmd.Run(); err != nil {
		var errorMsg string
		exitErr, isExitError := err.(*exec.ExitError)
		if isExitError {
			errorMsg = fmt.Sprintf("The command failed with code=%d, stderr=%s", exitErr.ExitCode(), stdErr.String())
		} else {
			errorMsg = fmt.Sprintf("The command failed, stderr=%s", stdErr.String())
		}
		return errors.Wrapf(err, errorMsg)
	}

	return nil
}

// NewCommandHealthcheck creates a Command healthcheck from a logger and a configuration
func NewCommandHealthcheck(logger *zap.Logger, config *CommandHealthcheckConfiguration) *CommandHealthcheck {
	return &CommandHealthcheck{
		Base: Base{
			Logger: logger,
			Config: config,
		},
	}
}
