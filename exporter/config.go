package exporter

import (
	"cabourotte/healthcheck"

	"github.com/pkg/errors"
)

// HTTPConfiguration The configuration for the HTTP exporter.
type HTTPConfiguration struct {
	Host     string
	Path     string
	Port     uint32
	Protocol healthcheck.Protocol
}

// Configuration the main configuration for the exporter component
type Configuration struct {
	HTTP []HTTPConfiguration
}

// UnmarshalYAML parses the configuration of the http component from YAML.
func (c *HTTPConfiguration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type rawConfiguration HTTPConfiguration
	raw := rawConfiguration{}
	if err := unmarshal(&raw); err != nil {
		return errors.Wrap(err, "Unable to read HTTP exporter configuration")
	}
	if raw.Host == "" {
		return errors.New("Invalid Host for the HTTP exporter configuration")
	}
	if raw.Port == 0 {
		return errors.New("Invalid Port for the HTTP server")
	}
	*c = HTTPConfiguration(raw)
	return nil
}
