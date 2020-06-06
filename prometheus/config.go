package prometheus

import (
	"cabourotte/healthcheck"
)

// Configuration the configuration for prometheus
type Configuration struct {
	Listen    string
	Interval  healthcheck.Duration
	Namespace string
	Subsystem string
	Cert      string
	Key       string
	Cacert    string
}
