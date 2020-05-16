package reloader

import (
	"github.com/pkg/errors"

	"cabourotte/healthcheck"
	"cabourotte/http"
)

// Configuration the HTTP server configuration
type Configuration struct {
	HTTP       http.Configuration
	DNSChecks  []healthcheck.DNSHealthcheckConfiguration  `yaml:"dns_checks"`
	TCPChecks  []healthcheck.TCPHealthcheckConfiguration  `yaml:"tcp_checks"`
	HTTPChecks []healthcheck.HTTPHealthcheckConfiguration `yaml:"http_checks"`
}

// UnmarshalYAML Parse a configuration from YAML.
func (configuration *Configuration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type rawConfiguration Configuration
	raw := rawConfiguration{}
	if err := unmarshal(&raw); err != nil {
		return errors.Wrap(err, "Unable to read Cabourotte configuration")
	}
	*configuration = Configuration(raw)
	return nil
}
