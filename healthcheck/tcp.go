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
	Name        string
	Description string
	// can be an IP or a domain
	Target   string
	Port     uint
	Timeout  Duration
	Interval Duration
	OneOff   bool
}

// TCPHealthcheck defines a TCP healthcheck
type TCPHealthcheck struct {
	Logger     *zap.Logger
	Config     *TCPHealthcheckConfiguration
	ChanResult chan *Result
	URL        string

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

// Initialize the healthcheck.
func (h *TCPHealthcheck) Initialize() error {
	h.buildURL()
	return nil
}

// Start an Healthcheck, which will be periodically executed after a
// given interval of time
func (h *TCPHealthcheck) Start(chanResult chan *Result) error {
	h.LogInfo("Starting healthcheck")
	h.ChanResult = chanResult
	h.Tick = time.NewTicker(time.Duration(h.Config.Interval))
	h.t.Go(func() error {
		for {
			select {
			case <-h.Tick.C:
				err := h.Execute()
				result := NewResult(h, err)
				h.ChanResult <- result
			case <-h.t.Dying():
				return nil
			}
		}
	})
	return nil
}

// Stop an Healthcheck
func (h *TCPHealthcheck) Stop() error {
	h.LogInfo("Stopping healthcheck")
	h.Tick.Stop()
	h.t.Kill(nil)
	h.t.Wait()
	return nil

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
	ctx := h.t.Context(nil)
	dialer := net.Dialer{}
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(h.Config.Timeout))
	defer cancel()
	conn, err := dialer.DialContext(timeoutCtx, "tcp", h.URL)
	if err != nil {
		return errors.Wrapf(err, "TCP connection failed on %s", h.URL)
	}
	err = conn.Close()
	if err != nil {
		return errors.Wrapf(err, "Unable to close TCP connection")
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
func (h TCPHealthcheck) MarshalJSON() ([]byte, error) {
	return json.Marshal(h.Config)
}
