package http

import (
	"github.com/pkg/errors"

	"github.com/mcorbin/cabourotte/healthcheck"
)

type Configuration struct {
	Host     string
	Path     string
	Port     uint32
	Protocol healthcheck.Protocol
	Interval healthcheck.Duration `json:"interval"`
	Key      string               `json:"key,omitempty"`
	Cert     string               `json:"cert,omitempty"`
	Cacert   string               `json:"cacert,omitempty"`
	Insecure bool
}

type ResultPayload struct {
	CommandChecks []healthcheck.CommandHealthcheckConfiguration `json:"command-checks"`
	DNSChecks     []healthcheck.DNSHealthcheckConfiguration     `json:"dns-checks"`
	TCPChecks     []healthcheck.TCPHealthcheckConfiguration     `json:"tcp-checks"`
	HTTPChecks    []healthcheck.HTTPHealthcheckConfiguration    `json:"http-checks"`
	TLSChecks     []healthcheck.TLSHealthcheckConfiguration     `json:"tls-checks"`
}

// UnmarshalYAML Parse a configuration from YAML.
func (configuration *Configuration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type rawConfiguration Configuration
	raw := rawConfiguration{}
	if err := unmarshal(&raw); err != nil {
		return errors.Wrap(err, "Unable to read HTTP discovery configuration")
	}
	if raw.Host == "" {
		return errors.New("Invalid host for the HTTP exporter configuration")
	}
	if raw.Port == 0 {
		return errors.New("Invalid port for the HTTP server")
	}
	if raw.Interval < 10 {
		return errors.New("The interval should be greater or equal than 10")
	}
	if !((raw.Key != "" && raw.Cert != "") ||
		(raw.Key == "" && raw.Cert == "")) {
		return errors.New("Invalid certificates")
	}
	*configuration = Configuration(raw)
	return nil
}
