package http

import (
	"net"

	"github.com/pkg/errors"
)

// BasicAuth basic auth for the configuration
type BasicAuth struct {
	Username string
	Password string
}

// Configuration the HTTP server configuration
type Configuration struct {
	Host      string
	Port      uint32
	BasicAuth BasicAuth `yaml:"basic-auth"`
}

// UnmarshalYAML parses the configuration of the http component from YAML.
func (c *Configuration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type rawConfiguration Configuration
	raw := rawConfiguration{}
	if err := unmarshal(&raw); err != nil {
		return errors.Wrap(err, "Unable to read HTTP configuration")
	}
	ip := net.ParseIP(raw.Host)
	if ip == nil {
		return errors.New("Invalid IP address for the HTTP server")
	}
	if raw.Port == 0 {
		return errors.New("Invalid Port for the HTTP server")
	}
	// if raw.BasicAuth.Username == "" || raw.BasicAuth.Password == "" {
	// 	return errors.New("Invalid Basic Auth configuration")
	// }
	*c = Configuration(raw)
	return nil
}
