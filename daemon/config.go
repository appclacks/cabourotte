package daemon

import (
	"github.com/pkg/errors"

	"cabourotte/exporter"
	"cabourotte/healthcheck"
	"cabourotte/http"
)

// Configuration the HTTP server configuration
type Configuration struct {
	ResultBuffer uint `yaml:"result_buffer"`
	HTTP         http.Configuration
	DNSChecks    []healthcheck.DNSHealthcheckConfiguration  `yaml:"dns_checks"`
	TCPChecks    []healthcheck.TCPHealthcheckConfiguration  `yaml:"tcp_checks"`
	HTTPChecks   []healthcheck.HTTPHealthcheckConfiguration `yaml:"http_checks"`
	Exporters    exporter.Configuration
}

const DefaultBufferSize = 5000

// UnmarshalYAML Parse a configuration from YAML.
func (configuration *Configuration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	chanSize := uint(DefaultBufferSize)
	type rawConfiguration Configuration
	raw := rawConfiguration{}
	if err := unmarshal(&raw); err != nil {
		return errors.Wrap(err, "Unable to read Cabourotte configuration")
	}
	for _, check := range raw.DNSChecks {
		err := healthcheck.ValidateDNSConfig(&check)
		if err != nil {
			return errors.Wrap(err, "Invalid healthcheck configuration")
		}
	}
	for _, check := range raw.TCPChecks {
		err := healthcheck.ValidateTCPConfig(&check)
		if err != nil {
			return errors.Wrap(err, "Invalid healthcheck configuration")
		}
	}
	for _, check := range raw.HTTPChecks {
		err := healthcheck.ValidateHTTPConfig(&check)
		if err != nil {
			return errors.Wrap(err, "Invalid healthcheck configuration")
		}
	}
	if raw.ResultBuffer == 0 {
		raw.ResultBuffer = chanSize
	}
	*configuration = Configuration(raw)
	return nil
}
