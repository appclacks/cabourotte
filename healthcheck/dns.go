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
	Config *DNSHealthcheckConfiguration
	URL    string

	Tick *time.Ticker
	t    tomb.Tomb
}

// Initialize the healthcheck.
func (h *DNSHealthcheck) Initialize() error {
	return nil
}

// Identifier returns the healthcheck identifier.
func (h *DNSHealthcheck) Identifier() string {
	return h.Config.Name
}

// Start an Healthcheck, which will be periodically executed after a
// given interval of time
func (h *DNSHealthcheck) Start() error {
	h.LogInfo("Starting healthcheck")
	h.Tick = time.NewTicker(time.Duration(h.Config.Interval))
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

// LogError logs an error with context
func (h *DNSHealthcheck) LogError(err error, message string) {
	h.Logger.Error(err.Error(),
		zap.String("extra", message),
		zap.String("domain", h.Config.Domain),
		zap.String("name", h.Config.Name))
}

// LogDebug logs a message with context
func (h *DNSHealthcheck) LogDebug(message string) {
	h.Logger.Debug(message,
		zap.String("domain", h.Config.Domain),
		zap.String("name", h.Config.Name))
}

// LogInfo logs a message with context
func (h *DNSHealthcheck) LogInfo(message string) {
	h.Logger.Info(message,
		zap.String("domain", h.Config.Domain),
		zap.String("name", h.Config.Name))
}

// Stop an Healthcheck
func (h *DNSHealthcheck) Stop() error {
	h.LogInfo("Stopping healthcheck")
	h.Tick.Stop()
	h.t.Kill(nil)
	h.t.Wait()
	return nil

}

// Execute executes an healthcheck on the given domain
func (h *DNSHealthcheck) Execute() error {
	h.LogDebug("start executing healthcheck")
	_, err := net.LookupIP(h.Config.Domain)
	if err != nil {
		return errors.Wrapf(err, "Fail to lookup IP for domain")
	}
	return nil
}

// NewDNSHealthcheck creates a DNS healthcheck from a logger and a configuration
func NewDNSHealthcheck(logger *zap.Logger, config *DNSHealthcheckConfiguration) DNSHealthcheck {
	return DNSHealthcheck{
		Logger: logger,
		Config: config,
	}
}
