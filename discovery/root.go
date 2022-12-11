package discovery

import (
	"fmt"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	dhttp "github.com/mcorbin/cabourotte/discovery/http"
	"github.com/mcorbin/cabourotte/healthcheck"
	"github.com/mcorbin/cabourotte/prometheus"
	prom "github.com/prometheus/client_golang/prometheus"
)

// Component contains all service discovery instances
type Component struct {
	Logger           *zap.Logger
	HTTPDiscovery    []*dhttp.HTTPDiscovery
	requestHistogram *prom.HistogramVec
	responseCounter  *prom.CounterVec
	Prometheus       *prometheus.Prometheus
}

// New creates the main component from its configuration
func New(logger *zap.Logger, config Configuration, promComponent *prometheus.Prometheus, healthcheck *healthcheck.Component) (*Component, error) {
	component := &Component{
		Logger: logger,
	}
	if len(config.HTTP) != 0 {
		buckets := []float64{
			0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 0.75, 1,
			2.5, 5, 7.5, 10}
		histo := prom.NewHistogramVec(prom.HistogramOpts{
			Name:    "http_discovery_duration_seconds",
			Help:    "Time to execute the HTTP request for healthchecks discovery.",
			Buckets: buckets,
		},
			[]string{"name"},
		)
		counter := prom.NewCounterVec(
			prom.CounterOpts{
				Name: "http_discovery_responses_total",
				Help: "Count the number of HTTP responses for discovery requests.",
			},
			[]string{"status", "name"})
		err := promComponent.Register(histo)
		if err != nil {
			return nil, errors.Wrapf(err, "fail to register the http discovery request histogram")
		}
		err = promComponent.Register(counter)
		if err != nil {
			return nil, errors.Wrapf(err, "fail to register the http discovery response counter")
		}
		httpNames := make(map[string]bool)
		var discovery []*dhttp.HTTPDiscovery
		for i := range config.HTTP {
			configHTTP := config.HTTP[i]
			_, ok := httpNames[configHTTP.Name]
			if ok {
				return nil, fmt.Errorf("HTTP discovery sources names should be unique (duplicate found for %s)", configHTTP.Name)
			}
			logger.Info(fmt.Sprintf("Enabling HTTP discovery %s", configHTTP.Name))
			httpDiscovery, err := dhttp.New(logger, &configHTTP, healthcheck, counter, histo)
			if err != nil {
				return nil, errors.Wrapf(err, "Fail to create the HTTP discovery component")
			}
			httpNames[configHTTP.Name] = true
			discovery = append(discovery, httpDiscovery)
		}
		component.HTTPDiscovery = discovery
		component.responseCounter = counter
		component.requestHistogram = histo
	}
	return component, nil
}

// Start start all discovery mechanisms
func (c *Component) Start() error {
	if c.HTTPDiscovery != nil && len(c.HTTPDiscovery) != 0 {
		for i := range c.HTTPDiscovery {
			discovery := c.HTTPDiscovery[i]
			err := discovery.Start()
			if err != nil {
				return err
			}
		}
	}
	return nil
}

// Stop stop all discovery mechanisms
func (c *Component) Stop() error {
	if c.HTTPDiscovery != nil && len(c.HTTPDiscovery) != 0 {
		for i := range c.HTTPDiscovery {
			discovery := c.HTTPDiscovery[i]
			err := discovery.Stop()
			if err != nil {
				return err
			}
		}
	}
	return nil
}
