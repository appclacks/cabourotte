package http

import (
	"net"

	"github.com/pkg/errors"
)

// Configuration the HTTP server configuration
type Configuration struct {
	Host string
	Port uint32
	Cert string
	Key  string
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
	if (raw.Cert != "" && raw.Key == "") || (raw.Cert == "" && raw.Key != "") {
		return errors.New("The cert and key options should be configured together")
	}
	*c = Configuration(raw)
	return nil
}
