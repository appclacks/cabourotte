package http

import (
	"net"

	"github.com/pkg/errors"
)

// Configuration the HTTP server configuration
type Configuration struct {
	Host string
	Port uint32
}

// UnmarshalYAML parses the configuration of the http component from YAML.
func (c *Configuration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	config := Configuration{}
	if err := unmarshal(&config); err != nil {
		return errors.Wrap(err, "Unable to read http configuration")
	}
	ip := net.ParseIP(config.Host)
	if ip == nil {
		return errors.New("Invalid IP address for the HTTP server")
	}
	*c = Configuration(config)
	return nil
}
