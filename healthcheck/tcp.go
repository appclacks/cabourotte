package healthcheck

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"gopkg.in/tomb.v2"
)

// TCPHealthcheckConfiguration defines a TCP healthcheck configuration
type TCPHealthcheckConfiguration struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	// can be an IP or a domain
	Target     string            `json:"target"`
	Port       uint              `json:"port"`
	SourceIP   IP                `json:"source-ip,omitempty" yaml:"source-ip,omitempty"`
	Timeout    Duration          `json:"timeout"`
	Interval   Duration          `json:"interval"`
	OneOff     bool              `json:"one-off"`
	ShouldFail bool              `json:"should-fail" yaml:"should-fail"`
	Labels     map[string]string `json:"labels,omitempty"`
}

// Validate validates the healthcheck configuration
func (config *TCPHealthcheckConfiguration) Validate() error {
	if config.Name == "" {
		return errors.New("The healthcheck name is missing")
	}
	if config.Target == "" {
		return errors.New("The healthcheck target is missing")
	}
	if config.Port == 0 {
		return errors.New("The healthcheck port is missing")
	}
	if config.Timeout == 0 {
		return errors.New("The healthcheck timeout is missing")
	}
	if !config.OneOff {
		if config.Interval < Duration(2*time.Second) {
			return errors.New("The healthcheck interval should be greater than 2 second")
		}
		if config.Interval < config.Timeout {
			return errors.New("The healthcheck interval should be greater than the timeout")
		}
	}
	return nil
}

// GetLabels returns the labels
func (h *TCPHealthcheck) GetLabels() map[string]string {
	return h.Config.Labels
}

// TCPHealthcheck defines a TCP healthcheck
type TCPHealthcheck struct {
	Logger *zap.Logger
	Config *TCPHealthcheckConfiguration
	URL    string

	Tick *time.Ticker
	t    tomb.Tomb
}

// buildURL build the target URL for the TCP healthcheck, depending of its
// configuration
func (h *TCPHealthcheck) buildURL() {
	h.URL = net.JoinHostPort(h.Config.Target, fmt.Sprintf("%d", h.Config.Port))
}

// Name returns the healthcheck identifier.
func (h *TCPHealthcheck) Name() string {
	return h.Config.Name
}

// Summary returns an healthcheck summary
func (h *TCPHealthcheck) Summary() string {
	summary := ""
	if h.Config.Description != "" {
		summary = fmt.Sprintf("%s on %s:%d", h.Config.Description, h.Config.Target, h.Config.Port)

	} else {
		summary = fmt.Sprintf("on %s:%d", h.Config.Target, h.Config.Port)
	}

	if h.Config.ShouldFail {
		summary = summary + ". This healthcheck has should-fail=true."
	}

	return summary
}

// Initialize the healthcheck.
func (h *TCPHealthcheck) Initialize() error {
	h.buildURL()
	return nil
}

// Interval Get the interval.
func (h *TCPHealthcheck) Interval() Duration {
	return h.Config.Interval
}

// GetConfig get the config
func (h *TCPHealthcheck) GetConfig() interface{} {
	return h.Config
}

// OneOff returns true if the healthcheck if a one-off check
func (h *TCPHealthcheck) OneOff() bool {
	return h.Config.OneOff

}

// LogError logs an error with context
func (h *TCPHealthcheck) LogError(err error, message string) {
	h.Logger.Error(err.Error(),
		zap.String("extra", message),
		zap.String("target", h.Config.Target),
		zap.Uint("port", h.Config.Port),
		zap.String("name", h.Config.Name))
}

// LogDebug logs a message with context
func (h *TCPHealthcheck) LogDebug(message string) {
	h.Logger.Debug(message,
		zap.String("target", h.Config.Target),
		zap.Uint("port", h.Config.Port),
		zap.String("name", h.Config.Name))
}

// LogInfo logs a message with context
func (h *TCPHealthcheck) LogInfo(message string) {
	h.Logger.Info(message,
		zap.String("target", h.Config.Target),
		zap.Uint("port", h.Config.Port),
		zap.String("name", h.Config.Name))
}

// Execute executes an healthcheck on the given target
func (h *TCPHealthcheck) Execute() error {
	h.LogDebug("start executing healthcheck")
	ctx := h.t.Context(context.TODO())
	dialer := net.Dialer{}
	if h.Config.SourceIP != nil {
		srcIP := net.IP(h.Config.SourceIP).String()
		addr, err := net.ResolveTCPAddr("tcp", fmt.Sprintf("%s:0", srcIP))
		if err != nil {
			return errors.Wrapf(err, "Fail to set the source IP %s", srcIP)
		}
		dialer = net.Dialer{
			LocalAddr: addr,
		}
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(h.Config.Timeout))
	defer cancel()
	conn, err := dialer.DialContext(timeoutCtx, "tcp", h.URL)
	if h.Config.ShouldFail {
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
		Logger: logger,
		Config: config,
	}
}

// MarshalJSON marshal to json a dns healthcheck
func (h *TCPHealthcheck) MarshalJSON() ([]byte, error) {
	return json.Marshal(h.Config)
}
