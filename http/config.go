package http

import (
	"fmt"
	"net"

	"github.com/pkg/errors"

	"github.com/mcorbin/cabourotte/healthcheck"
	"github.com/mcorbin/httpgo"
)

// BasicAuth basic auth for the configuration
type BasicAuth struct {
	Username string
	Password string
}

// Configuration the HTTP server configuration
type Configuration struct {
	HTTP                  http.Configuration `yaml:"inline" json:"inline"`
	DisableHealthcheckAPI bool               `yaml:"disable-healthcheck-api,omitempty"`
	DisableResultAPI      bool               `yaml:"disable-result-api,omitempty"`
	Key                   string
	Cert                  string
	AllowedCN             []string `yaml:"allowed-cn"`
	Cacert                string
}

// UnmarshalYAML parses the configuration of the http component from YAML.
func (c *Configuration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type rawConfiguration Configuration
	raw := rawConfiguration{}
	if err := unmarshal(&raw); err != nil {
		return errors.Wrap(err, "Unable to read HTTP configuration")
	}
	if (raw.Cert != "" && raw.Key == "") || (raw.Cert == "" && raw.Key != "") {
		return errors.New("The cert and key options should be configured together")
	}
	if !((raw.Key != "" && raw.Cert != "" && raw.Cacert != "") ||
		(raw.Key == "" && raw.Cert == "" && raw.Cacert == "")) {
		return errors.New("Invalid certificates")
	}
	if (raw.HTTP.BasicAuth.Username == "" && raw.HTTP.BasicAuth.Password != "") ||
		(raw.HTTP.BasicAuth.Username != "" && raw.HTTP.BasicAuth.Password == "") {
		return errors.New("Invalid Basic Auth configuration")
	}
	*c = Configuration(raw)
	return nil
}

// BulkPayload the paylaod for bulk requests fo healthchecks
type BulkPayload struct {
	DNSChecks     []healthcheck.DNSHealthcheckConfiguration     `json:"dns-checks"`
	CommandChecks []healthcheck.CommandHealthcheckConfiguration `json:"command-checks"`
	TCPChecks     []healthcheck.TCPHealthcheckConfiguration     `json:"tcp-checks"`
	HTTPChecks    []healthcheck.HTTPHealthcheckConfiguration    `json:"http-checks"`
	TLSChecks     []healthcheck.TLSHealthcheckConfiguration     `json:"tls-checks"`
}

// Validate validates the payload for bulk requests
func (p *BulkPayload) Validate() error {
	oneOffErrorMsg := "One-off healthchecks are not supported for bulk requests"
	for _, config := range p.DNSChecks {
		err := config.Validate()
		if config.Base.OneOff {
			return errors.New(oneOffErrorMsg)
		}
		if err != nil {
			msg := fmt.Sprintf("Invalid healthcheck configuration: %s", err.Error())
			return errors.New(msg)
		}
	}
	for _, config := range p.TCPChecks {
		err := config.Validate()
		if config.Base.OneOff {
			return errors.New(oneOffErrorMsg)
		}
		if err != nil {
			msg := fmt.Sprintf("Invalid healthcheck configuration: %s", err.Error())
			return errors.New(msg)
		}
	}
	for _, config := range p.HTTPChecks {
		err := config.Validate()
		if config.Base.OneOff {
			return errors.New(oneOffErrorMsg)
		}
		if err != nil {
			msg := fmt.Sprintf("Invalid healthcheck configuration: %s", err.Error())
			return errors.New(msg)
		}
	}
	for _, config := range p.TLSChecks {
		err := config.Validate()
		if config.Base.OneOff {
			return errors.New(oneOffErrorMsg)
		}
		if err != nil {
			msg := fmt.Sprintf("Invalid healthcheck configuration: %s", err.Error())
			return errors.New(msg)
		}
	}
	for _, config := range p.CommandChecks {
		err := config.Validate()
		if config.Base.OneOff {
			return errors.New(oneOffErrorMsg)
		}
		if err != nil {
			msg := fmt.Sprintf("Invalid healthcheck configuration: %s", err.Error())
			return errors.New(msg)
		}
	}
	return nil
}
