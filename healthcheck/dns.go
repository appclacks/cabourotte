package healthcheck

import (
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"net"

	"gopkg.in/tomb.v2"
)

// DNSHealthcheckConfiguration defines a DNS healthcheck configuration
type DNSHealthcheckConfiguration struct {
	Name        string
	Description string
	Domain      string
	Interval    time.Duration
	OneOff      bool
}

// DNSHealthcheck defines an HTTP healthcheck
type DNSHealthcheck struct {
	Logger *zap.Logger
	ID     string
	config *DNSHealthcheckConfiguration
	URL    string

	Tick *time.Ticker
	t    tomb.Tomb
}

// Initialize the healthcheck.
func (h *DNSHealthcheck) Initialize() error {
	return nil
}

// Start an Healthcheck, which will be periodically executed after a
// given interval of time
func (h *DNSHealthcheck) Start() error {
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

// logError logs an error with context
func (h *DNSHealthcheck) LogError(err error, message string) {
	h.Logger.Error(err.Error(),
		zap.String("extra", message),
		zap.String("domain", h.config.Domain),
		zap.String("name", h.config.Name),
		zap.String("id", h.ID))
}

// logError logs a message with context
func (h *DNSHealthcheck) LogDebug(message string) {
	h.Logger.Debug(message,
		zap.String("domain", h.config.Domain),
		zap.String("name", h.config.Name),
		zap.String("id", h.ID))
}

// Stop an Healthcheck
func (h *DNSHealthcheck) Stop() error {
	h.Tick.Stop()
	h.t.Kill(nil)
	h.t.Wait()
	return nil

}

// Execute executes an healthcheck on the given domain
func (h *DNSHealthcheck) Execute() error {
	h.LogDebug("start executing healthcheck")
	_, err := net.LookupIP(h.config.Domain)
	if err != nil {
		return errors.Wrapf(err, "Fail to lookup IP for domain")
	}
	return nil
}
