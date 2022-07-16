package discovery

import (
	"github.com/pkg/errors"
	"go.uber.org/zap"

	dhttp "github.com/mcorbin/cabourotte/discovery/http"
	"github.com/mcorbin/cabourotte/healthcheck"
	"github.com/mcorbin/cabourotte/prometheus"
)

// Component contains all service discovery instances
type Component struct {
	Logger        *zap.Logger
	HTTPDiscovery *dhttp.HTTPDiscovery
	Prometheus    *prometheus.Prometheus
}

// New creates the main component from its configuration
func New(logger *zap.Logger, config Configuration, promComponent *prometheus.Prometheus, healthcheck *healthcheck.Component) (*Component, error) {
	component := &Component{
		Logger: logger,
	}
	if config.HTTP.Host != "" {
		logger.Info("Enabling HTTP discovery")
		httpDiscovery, err := dhttp.New(logger, &config.HTTP, healthcheck, promComponent)
		if err != nil {
			return nil, errors.Wrapf(err, "Fail to create the HTTP discovery component")
		}
		component.HTTPDiscovery = httpDiscovery
	}
	return component, nil
}

// Start start all discovery mechanisms
func (c *Component) Start() error {
	if c.HTTPDiscovery != nil {
		err := c.HTTPDiscovery.Start()
		if err != nil {
			return err
		}
	}
	return nil
}

// Stop stop all discovery mechanisms
func (c *Component) Stop() error {
	if c.HTTPDiscovery != nil {
		err := c.HTTPDiscovery.Stop()
		if err != nil {
			return err
		}
	}
	return nil
}
