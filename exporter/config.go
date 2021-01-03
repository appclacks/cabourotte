package exporter

import (
	"cabourotte/healthcheck"

	"github.com/pkg/errors"
)

// HTTPConfiguration The configuration for the HTTP exporter.
type HTTPConfiguration struct {
	Name     string
	Host     string
	Path     string
	Port     uint32
	Protocol healthcheck.Protocol
	Key      string `json:"key,omitempty"`
	Cert     string `json:"cert,omitempty"`
	Cacert   string `json:"cacert,omitempty"`
	Insecure bool
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
		return errors.New("Invalid host for the HTTP exporter configuration")
	}
	if raw.Name == "" {
		return errors.New("Invalid name for the HTTP exporter configuration")
	}
	if raw.Port == 0 {
		return errors.New("Invalid port for the HTTP server")
	}
	if !((raw.Key != "" && raw.Cert != "") ||
		(raw.Key == "" && raw.Cert == "")) {
		return errors.New("Invalid certificates")
	}
	*c = HTTPConfiguration(raw)
	return nil
}
