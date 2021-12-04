package client

import (
	"github.com/pkg/errors"

	"github.com/mcorbin/cabourotte/healthcheck"
	"github.com/mcorbin/cabourotte/http"
)

// Configuration of a Cabourotte client
type Configuration struct {
	Host      string
	Port      uint32
	Path      string
	Protocol  healthcheck.Protocol
	Key       string
	Cert      string
	Cacert    string
	BasicAuth http.BasicAuth `yaml:"basic-auth" json:"basic-auth"`
	Insecure  bool
}

// UnmarshalYAML parses the configuration.
func (c *Configuration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type rawConfiguration Configuration
	raw := rawConfiguration{}
	if err := unmarshal(&raw); err != nil {
		return errors.Wrap(err, "Unable to read HTTP exporter configuration")
	}
	if raw.Host == "" {
		return errors.New("Invalid host for the HTTP exporter configuration")
	}
	if raw.Port == 0 {
		return errors.New("Invalid port for the HTTP server")
	}
	if !((raw.Key != "" && raw.Cert != "") ||
		(raw.Key == "" && raw.Cert == "")) {
		return errors.New("Invalid certificates")
	}
	*c = Configuration(raw)
	return nil
}
