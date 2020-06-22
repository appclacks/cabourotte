package healthcheck

import (
	"encoding/json"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
	"net"

	"gopkg.in/tomb.v2"
)

// DNSHealthcheckConfiguration defines a DNS healthcheck configuration
type DNSHealthcheckConfiguration struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Domain      string   `json:"domain"`
	Interval    Duration `json:"interval"`
	OneOff      bool     `json:"one-off,"`
}

// DNSHealthcheck defines an HTTP healthcheck
type DNSHealthcheck struct {
	Logger *zap.Logger
	Config *DNSHealthcheckConfiguration
	URL    string

	Tick *time.Ticker
	t    tomb.Tomb
}

// ValidateDNSConfig validates the healthcheck configuration
func ValidateDNSConfig(config *DNSHealthcheckConfiguration) error {
	if config.Name == "" {
		return errors.New("The healthcheck name is missing")
	}
	if config.Domain == "" {
		return errors.New("The healthcheck domain is missing")
	}
	if config.Interval < Duration(2*time.Second) {
		return errors.New("The healthcheck interval should be greater than 2 second")
	}
	return nil
}

// Initialize the healthcheck.
func (h *DNSHealthcheck) Initialize() error {
	return nil
}

// Interval Get the interval.
func (h *DNSHealthcheck) Interval() Duration {
	return h.Config.Interval
}

// GetConfig get the config
func (h *DNSHealthcheck) GetConfig() interface{} {
	return h.Config
}

// Name returns the healthcheck identifier.
func (h *DNSHealthcheck) Name() string {
	return h.Config.Name
}

// OneOff returns true if the healthcheck if a one-off check
func (h *DNSHealthcheck) OneOff() bool {
	return h.Config.OneOff

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
func NewDNSHealthcheck(logger *zap.Logger, config *DNSHealthcheckConfiguration) *DNSHealthcheck {
	return &DNSHealthcheck{
		Logger: logger,
		Config: config,
	}
}

// MarshalJSON marshal to json a dns healthcheck
func (h *DNSHealthcheck) MarshalJSON() ([]byte, error) {
	return json.Marshal(h.Config)
}
