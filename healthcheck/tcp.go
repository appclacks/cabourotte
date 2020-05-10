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
	Name        string
	Description string
	// can be an IP or a domain
	Target   string
	Port     uint
	Timeout  time.Duration
	Interval time.Duration
	OneOff   bool
}

// TCPHealthcheck defines a TCP healthcheck
type TCPHealthcheck struct {
	Logger *zap.Logger
	config *TCPHealthcheckConfiguration
	URL    string

	Tick *time.Ticker
	t    tomb.Tomb
}

// buildURL build the target URL for the TCP healthcheck, depending of its
// configuration
func (h *TCPHealthcheck) buildURL() {
	h.URL = net.JoinHostPort(h.config.Target, fmt.Sprintf("%d", h.config.Port))
}

// Identifier returns the healthcheck identifier.
func (h *TCPHealthcheck) Identifier() string {
	return h.config.Name
}

// Initialize the healthcheck.
func (h *TCPHealthcheck) Initialize() error {
	h.buildURL()
	return nil
}

// Start an Healthcheck, which will be periodically executed after a
// given interval of time
func (h *TCPHealthcheck) Start() error {
	h.LogInfo("Starting healthcheck")
	h.Tick = time.NewTicker(time.Duration(h.config.Interval))
	h.t.Go(func() error {
		for {
			select {
			case <-h.Tick.C:
				h.Execute()
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
		zap.String("target", h.config.Target),
		zap.Uint("port", h.config.Port),
		zap.String("name", h.config.Name))
}

// LogDebug logs a message with context
func (h *TCPHealthcheck) LogDebug(message string) {
	h.Logger.Debug(message,
		zap.String("target", h.config.Target),
		zap.Uint("port", h.config.Port),
		zap.String("name", h.config.Name))
}

// LogInfo logs a message with context
func (h *TCPHealthcheck) LogInfo(message string) {
	h.Logger.Info(message,
		zap.String("target", h.config.Target),
		zap.Uint("port", h.config.Port),
		zap.String("name", h.config.Name))
}

// Execute executes an healthcheck on the given target
func (h *TCPHealthcheck) Execute() error {
	h.LogDebug("start executing healthcheck")
	ctx := h.t.Context(nil)
	dialer := net.Dialer{}
	timeoutCtx, cancel := context.WithTimeout(ctx, h.config.Timeout)
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
