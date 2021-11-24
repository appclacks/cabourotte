package daemon

import (
	"github.com/pkg/errors"

	"cabourotte/exporter"
	"cabourotte/healthcheck"
	"cabourotte/http"
)

// Configuration the HTTP server configuration
type Configuration struct {
	ResultBuffer  uint `yaml:"result-buffer"`
	HTTP          http.Configuration
	CommandChecks []healthcheck.CommandHealthcheckConfiguration `yaml:"command-checks"`
	DNSChecks     []healthcheck.DNSHealthcheckConfiguration     `yaml:"dns-checks"`
	TCPChecks     []healthcheck.TCPHealthcheckConfiguration     `yaml:"tcp-checks"`
	HTTPChecks    []healthcheck.HTTPHealthcheckConfiguration    `yaml:"http-checks"`
	TLSChecks     []healthcheck.TLSHealthcheckConfiguration     `yaml:"tls-checks"`
	Exporters     exporter.Configuration
}

// DefaultBufferSize the default siez for the buffer containing healthchecks results
const DefaultBufferSize = 5000

// configChecksNames returns a map containing the checks managed by the
// configuration file
func (configuration *Configuration) configChecksNames() map[string]bool {
	checks := make(map[string]bool)
	for i := range configuration.CommandChecks {
		checks[configuration.CommandChecks[i].Base.Name] = true
	}
	for i := range configuration.DNSChecks {
		checks[configuration.DNSChecks[i].Base.Name] = true
	}
	for i := range configuration.TCPChecks {
		checks[configuration.TCPChecks[i].Base.Name] = true
	}
	for i := range configuration.HTTPChecks {
		checks[configuration.HTTPChecks[i].Base.Name] = true
	}
	for i := range configuration.TLSChecks {
		checks[configuration.TLSChecks[i].Base.Name] = true
	}
	return checks
}

// UnmarshalYAML Parse a configuration from YAML.
func (configuration *Configuration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	chanSize := uint(DefaultBufferSize)
	type rawConfiguration Configuration
	raw := rawConfiguration{}
	if err := unmarshal(&raw); err != nil {
		return errors.Wrap(err, "Unable to read Cabourotte configuration")
	}
	for i := range raw.CommandChecks {
		check := raw.CommandChecks[i]
		err := check.Validate()
		if err != nil {
			return errors.Wrap(err, "Invalid healthcheck configuration")
		}
	}
	for i := range raw.DNSChecks {
		check := raw.DNSChecks[i]
		err := check.Validate()
		if err != nil {
			return errors.Wrap(err, "Invalid healthcheck configuration")
		}
	}
	for i := range raw.TCPChecks {
		check := raw.TCPChecks[i]
		err := check.Validate()
		if err != nil {
			return errors.Wrap(err, "Invalid healthcheck configuration")
		}
	}
	for i := range raw.HTTPChecks {
		check := raw.HTTPChecks[i]
		err := check.Validate()
		if err != nil {
			return errors.Wrap(err, "Invalid healthcheck configuration")
		}
	}
	for i := range raw.TLSChecks {
		check := raw.TLSChecks[i]
		err := check.Validate()
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
