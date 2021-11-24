package healthcheck

// Base shared fields between healthchecks
type Base struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Interval    Duration          `json:"interval"`
	OneOff      bool              `json:"one-off"`
	Labels      map[string]string `json:"labels,omitempty"`
}
