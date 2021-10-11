package healthcheck

import (
	"fmt"

	"net"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// DNSHealthcheckConfiguration defines a DNS healthcheck configuration
type DNSHealthcheckConfiguration struct {
	BaseConfig  `json:",inline"`
	ExpectedIPs []IP   `json:"expected-ips,omitempty" yaml:"expected-ips,omitempty"`
	Domain      string `json:"domain"`
	// No Timeout
}

// DNSHealthcheck defines an HTTP healthcheck
type DNSHealthcheck struct {
	Base
}

// Validate validates the healthcheck configuration
func (config *DNSHealthcheckConfiguration) Validate() error {
	if err := config.BaseConfig.Validate(); err != nil {
		return err
	}
	if config.Domain == "" {
		return errors.New("The healthcheck domain is missing")
	}
	return nil
}

// Initialize the healthcheck.
func (h *DNSHealthcheck) Initialize() error {
	return nil
}

// Summary returns an healthcheck summary
func (h *DNSHealthcheck) Summary() string {
	summary := ""
	if h.Config.GetDescription() != "" {
		summary = fmt.Sprintf("%s on %s", h.Config.GetDescription(), h.Config.(*DNSHealthcheckConfiguration).Domain)

	} else {
		summary = fmt.Sprintf("on %s", h.Config.(*DNSHealthcheckConfiguration).Domain)
	}

	return summary
}

// LogError logs an error with context
func (h *DNSHealthcheck) LogError(err error, message string) {
	h.Logger.Error(err.Error(),
		zap.String("extra", message),
		zap.String("domain", h.Config.(*DNSHealthcheckConfiguration).Domain),
		zap.String("name", h.Config.GetName()))
}

// LogDebug logs a message with context
func (h *DNSHealthcheck) LogDebug(message string) {
	h.Logger.Debug(message,
		zap.String("domain", h.Config.(*DNSHealthcheckConfiguration).Domain),
		zap.String("name", h.Config.GetName()))
}

// LogInfo logs a message with context
func (h *DNSHealthcheck) LogInfo(message string) {
	h.Logger.Info(message,
		zap.String("domain", h.Config.(*DNSHealthcheckConfiguration).Domain),
		zap.String("name", h.Config.GetName()))
}

func verifyIPs(expectedIPs []IP, lookupIPs []net.IP) error {
	notFound := []string{}
	for i := range expectedIPs {
		netIP := net.IP(expectedIPs[i])
		found := false
		for j := range lookupIPs {
			respIP := lookupIPs[j]
			if netIP.Equal(respIP) {
				found = true
				break
			}
		}
		if !found {
			notFound = append(notFound, netIP.String())
		}
	}
	if len(notFound) != 0 {
		l := ""
		for _, notFound := range notFound {
			l = l + "," + notFound
		}
		return fmt.Errorf("Expected IP address not found. IPs found are %s", l)
	}
	return nil
}

// Execute executes an healthcheck on the given domain
func (h *DNSHealthcheck) Execute() error {
	h.LogDebug("start executing healthcheck")
	ips, err := net.LookupIP(h.Config.(*DNSHealthcheckConfiguration).Domain)
	if err != nil {
		return errors.Wrapf(err, "Fail to lookup IP for domain")
	}
	err = verifyIPs(h.Config.(*DNSHealthcheckConfiguration).ExpectedIPs, ips)
	if err != nil {
		return err
	}
	return nil
}

// NewDNSHealthcheck creates a DNS healthcheck from a logger and a configuration
func NewDNSHealthcheck(logger *zap.Logger, config *DNSHealthcheckConfiguration) *DNSHealthcheck {
	return &DNSHealthcheck{
		Base: Base{
			Logger: logger,
			Config: config,
		},
	}
}
