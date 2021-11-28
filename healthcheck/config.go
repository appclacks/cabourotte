package healthcheck

const (
	// SourceConfig the check is managed by the configuration file
	SourceConfig string = ""
	// SourceAPI the check is managed by the API
	SourceAPI string = "api"
	// SourceKubernetesPod the check was created from a Kubernetes pod
	SourceKubernetesPod string = "kubernetes-pod"
	// SourceKubernetesService the check was created from a service pod
	SourceKubernetesService string = "kubernetes-service"
)

// Base shared fields between healthchecks
type Base struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Interval    Duration          `json:"interval"`
	OneOff      bool              `json:"one-off"`
	Source      string            `json:"source"`
	Labels      map[string]string `json:"labels,omitempty"`
}

// SourceChecksNames returns all checks managed by the given source
func (c *Component) SourceChecksNames(source string) map[string]bool {
	c.lock.Lock()
	defer c.lock.Unlock()
	checks := make(map[string]bool)
	for i := range c.Healthchecks {
		wrapper := c.Healthchecks[i]
		if wrapper.healthcheck.Base().Source == source {
			checks[wrapper.healthcheck.Base().Name] = true
		}
	}
	return checks
}
