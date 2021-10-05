package prometheus

import (
	"net/http"

	prom "github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// Prometheus contains the prometheus registry and config
type Prometheus struct {
	Config   *Configuration
	Logger   *zap.Logger
	Registry *prom.Registry
}

// New creates a new Prometheus component
func New() (*Prometheus, error) {
	reg := prom.NewRegistry()
	p := &Prometheus{
		Registry: reg,
	}
	err := p.Register(collectors.NewGoCollector())
	if err != nil {
		return nil, err
	}
	return p, nil
}

// Register adds a metric to the component
func (p *Prometheus) Register(collector prom.Collector) error {
	return p.Registry.Register(collector)
}

// Unregister removes a metric from the component
func (p *Prometheus) Unregister(collector prom.Collector) {
	p.Registry.Unregister(collector)
}

// Handler returns the handler for the prometheus component
func (p *Prometheus) Handler() http.Handler {
	return promhttp.HandlerFor(p.Registry, promhttp.HandlerOpts{})
}
