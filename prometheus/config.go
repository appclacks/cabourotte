package prometheus

// Configuration the configuration for prometheus
type Configuration struct {
	Listen    string
	Namespace string
	Subsystem string
	Cert      string
	Key       string
	Cacert    string
}
