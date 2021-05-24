package exporter

// Configuration the main configuration for the exporter component
type Configuration struct {
	HTTP    []HTTPConfiguration
	Riemann []RiemannConfiguration
}
