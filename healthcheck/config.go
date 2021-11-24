package healthcheck

type Source int64

const (
	Config Source = iota
	API
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
