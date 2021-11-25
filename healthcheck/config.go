package healthcheck

// Source which component created the healthcheck
type Source string

const (
	// SourceConfig the check is managed by the configuration file
	SourceConfig Source = ""
	// SourceAPI the check is managed by the API
	SourceAPI Source = "api"
)

// Base shared fields between healthchecks
type Base struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Interval    Duration          `json:"interval"`
	OneOff      bool              `json:"one-off"`
	Source      Source            `json:"source"`
	Labels      map[string]string `json:"labels,omitempty"`
}

// SourceChecksNames returns all checks managed by the given source
func (c *Component) SourceChecksNames(source Source) map[string]bool {
	c.lock.RLock()
	defer c.lock.RUnlock()
	checks := make(map[string]bool)
	for i := range c.Healthchecks {
		wrapper := c.Healthchecks[i]
		if wrapper.healthcheck.Base().Source == source {
			checks[wrapper.healthcheck.Base().Name] = true
		}
	}
	return checks
}
