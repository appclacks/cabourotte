package healthcheck

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"gopkg.in/tomb.v2"
)

// TCPHealthcheckConfiguration defines a TCP healthcheck configuration
type TCPHealthcheckConfiguration struct {
	BaseConfig `json:",inline"`
	// can be an IP or a domain
	Target     string `json:"target"`
	Port       uint   `json:"port"`
	SourceIP   IP     `json:"source-ip,omitempty" yaml:"source-ip,omitempty"`
	ShouldFail bool   `json:"should-fail" yaml:"should-fail"`
}

// Validate validates the healthcheck configuration
func (config *TCPHealthcheckConfiguration) Validate() error {
	if err := config.BaseConfig.Validate(); err != nil {
		return err
	}
	if config.Target == "" {
		return errors.New("The healthcheck target is missing")
	}
	if config.Port == 0 {
		return errors.New("The healthcheck port is missing")
	}
	return nil
}

// TCPHealthcheck defines a TCP healthcheck
type TCPHealthcheck struct {
	Base
	t tomb.Tomb
}

// buildURL build the target URL for the TCP healthcheck, depending of its
// configuration
func (h *TCPHealthcheck) buildURL() {
	h.URL = net.JoinHostPort(h.Config.(*TCPHealthcheckConfiguration).Target, fmt.Sprintf("%d", h.Config.(*TCPHealthcheckConfiguration).Port))
}

// Summary returns an healthcheck summary
func (h *TCPHealthcheck) Summary() string {
	summary := ""
	if h.Config.GetDescription() != "" {
		summary = fmt.Sprintf("%s on %s:%d", h.Config.GetDescription(), h.Config.(*TCPHealthcheckConfiguration).Target, h.Config.(*TCPHealthcheckConfiguration).Port)

	} else {
		summary = fmt.Sprintf("on %s:%d", h.Config.(*TCPHealthcheckConfiguration).Target, h.Config.(*TCPHealthcheckConfiguration).Port)
	}

	if h.Config.(*TCPHealthcheckConfiguration).ShouldFail {
		summary = summary + ". This healthcheck has should-fail=true."
	}

	return summary
}

// Initialize the healthcheck.
func (h *TCPHealthcheck) Initialize() error {
	h.buildURL()
	return nil
}

// LogError logs an error with context
func (h *TCPHealthcheck) LogError(err error, message string) {
	h.Logger.Error(err.Error(),
		zap.String("extra", message),
		zap.String("target", h.Config.(*TCPHealthcheckConfiguration).Target),
		zap.Uint("port", h.Config.(*TCPHealthcheckConfiguration).Port),
		zap.String("name", h.Config.GetName()))
}

// LogDebug logs a message with context
func (h *TCPHealthcheck) LogDebug(message string) {
	h.Logger.Debug(message,
		zap.String("target", h.Config.(*TCPHealthcheckConfiguration).Target),
		zap.Uint("port", h.Config.(*TCPHealthcheckConfiguration).Port),
		zap.String("name", h.Config.GetName()))
}

// LogInfo logs a message with context
func (h *TCPHealthcheck) LogInfo(message string) {
	h.Logger.Info(message,
		zap.String("target", h.Config.(*TCPHealthcheckConfiguration).Target),
		zap.Uint("port", h.Config.(*TCPHealthcheckConfiguration).Port),
		zap.String("name", h.Config.GetName()))
}

// Execute executes an healthcheck on the given target
func (h *TCPHealthcheck) Execute() error {
	h.LogDebug("start executing healthcheck")
	ctx := h.t.Context(context.TODO())
	dialer := net.Dialer{}
	if h.Config.(*TCPHealthcheckConfiguration).SourceIP != nil {
		srcIP := net.IP(h.Config.(*TCPHealthcheckConfiguration).SourceIP).String()
		addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:0", srcIP))
		if err != nil {
			return errors.Wrapf(err, "Fail to set the source IP %s", srcIP)
		}
		dialer = net.Dialer{
			LocalAddr: addr,
		}
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(h.Config.(*TCPHealthcheckConfiguration).Timeout))
	defer cancel()
	conn, err := dialer.DialContext(timeoutCtx, "tcp", h.URL)
	if h.Config.(*TCPHealthcheckConfiguration).ShouldFail {
		if err == nil {
			defer conn.Close()
			return fmt.Errorf("TCP check is successful on %s but an error was expected", h.URL)
		}
	} else {
		if err != nil {
			return errors.Wrapf(err, "TCP connection failed on %s", h.URL)
		}
		defer conn.Close()
	}
	return nil
}

// NewTCPHealthcheck creates a TCP healthcheck from a logger and a configuration
func NewTCPHealthcheck(logger *zap.Logger, config *TCPHealthcheckConfiguration) *TCPHealthcheck {
	return &TCPHealthcheck{
		Base: Base{
			Logger: logger,
			Config: config,
		},
	}
}
