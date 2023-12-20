package daemon

import (
	"reflect"
	"sync"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/appclacks/cabourotte/discovery"
	"github.com/appclacks/cabourotte/exporter"
	"github.com/appclacks/cabourotte/healthcheck"
	"github.com/appclacks/cabourotte/http"
	"github.com/appclacks/cabourotte/memorystore"
	"github.com/appclacks/cabourotte/prometheus"
)

// Component is the component which will manage the HTTP server and the program
// configuration
type Component struct {
	Config      *Configuration
	MemoryStore *memorystore.MemoryStore
	Logger      *zap.Logger
	HTTP        *http.Component
	Healthcheck *healthcheck.Component
	Exporter    *exporter.Component
	Prometheus  *prometheus.Prometheus
	Discovery   *discovery.Component
	lock        sync.RWMutex
	ChanResult  chan *healthcheck.Result
}

// New creates and start a new daemon component
func New(logger *zap.Logger, config *Configuration) (*Component, error) {
	logger.Info("Starting the Cabourotte daemon")
	prom, err := prometheus.New()
	if err != nil {
		return nil, err
	}
	chanResult := make(chan *healthcheck.Result, config.ResultBuffer)
	checkComponent, err := healthcheck.New(logger, chanResult, prom, config.HealthchecksLabels)
	if err != nil {
		return nil, errors.Wrapf(err, "Fail to create the healthcheck component")
	}
	memstore := memorystore.NewMemoryStore(logger)
	memstore.Start()
	err = checkComponent.Start()
	if err != nil {
		return nil, errors.Wrapf(err, "Fail to start the healthcheck component")
	}
	http, err := http.New(logger, memstore, prom, &config.HTTP, checkComponent)
	if err != nil {
		return nil, errors.Wrapf(err, "Fail to create the HTTP server")
	}
	err = http.Start()
	if err != nil {
		return nil, errors.Wrapf(err, "Fail to start the HTTP server")
	}
	exporterComponent, err := exporter.New(logger, memstore, chanResult, prom, &config.Exporters)
	if err != nil {
		return nil, errors.Wrapf(err, "Fail to create the exporter component")
	}
	err = exporterComponent.Start()
	if err != nil {
		return nil, errors.Wrapf(err, "Fail to start the exporter component")
	}
	discoveryComponent, err := discovery.New(logger, config.Discovery, prom, checkComponent)
	if err != nil {
		return nil, errors.Wrapf(err, "Fail to create the service discovery component")
	}
	err = discoveryComponent.Start()
	if err != nil {
		return nil, errors.Wrapf(err, "Fail to start the service discovery component")
	}
	component := Component{
		MemoryStore: memstore,
		ChanResult:  chanResult,
		Config:      config,
		Prometheus:  prom,
		HTTP:        http,
		Logger:      logger,
		Exporter:    exporterComponent,
		Discovery:   discoveryComponent,
		Healthcheck: checkComponent,
	}
	err = component.ReloadHealthchecks(config)
	if err != nil {
		return nil, err
	}
	return &component, nil
}

// Stop stops the Cabourotte daemon
func (c *Component) Stop() error {
	c.Logger.Info("Stopping the Cabourotte daemon")
	c.lock.Lock()
	defer c.lock.Unlock()
	err := c.Discovery.Stop()
	if err != nil {
		return errors.Wrapf(err, "Fail to stop the service discovery component")
	}
	err = c.HTTP.Stop()
	if err != nil {
		return errors.Wrapf(err, "Fail to stop the HTTP server")
	}
	err = c.Healthcheck.Stop()
	if err != nil {
		return errors.Wrapf(err, "Fail to stop the healthcheck component")
	}
	close(c.ChanResult)
	err = c.Exporter.Stop()
	if err != nil {
		return errors.Wrapf(err, "Fail to stop the exporter component")
	}
	return nil
}

// ReloadHealthchecks reloads the healthchecks from a configuration
func (c *Component) ReloadHealthchecks(daemonConfig *Configuration) error {
	return c.Healthcheck.ReloadForSource(
		healthcheck.SourceConfig,
		nil,
		daemonConfig.CommandChecks,
		daemonConfig.DNSChecks,
		daemonConfig.TCPChecks,
		daemonConfig.HTTPChecks,
		daemonConfig.TLSChecks)
}

// Reload reloads the Cabourotte daemon. This function will remove or keep
// existing healthchecks depending of the new configuration. New checks will be added.
// The HTTP server will also be reloaded if its configuration has changed.
func (c *Component) Reload(daemonConfig *Configuration) error {
	c.Logger.Info("Reloading the Cabourotte daemon")
	c.lock.Lock()
	defer c.lock.Unlock()
	err := c.ReloadHealthchecks(daemonConfig)
	if err != nil {
		return errors.Wrapf(err, "Fail to reload healthchecks")
	}
	// compare the server config to see if we need to recreate it
	if !reflect.DeepEqual(c.Config.HTTP, daemonConfig.HTTP) {
		err := c.HTTP.Stop()
		if err != nil {
			return errors.Wrapf(err, "Fail to stop the HTTP server")
		}
		http, err := http.New(c.Logger, c.MemoryStore, c.Prometheus, &daemonConfig.HTTP, c.Healthcheck)
		if err != nil {
			return errors.Wrapf(err, "Fail to create the HTTP server")
		}
		err = http.Start()
		if err != nil {
			return errors.Wrapf(err, "Fail to start the HTTP server")
		}
		c.HTTP = http
	}
	c.Config = daemonConfig
	c.Logger.Info("Reloaded")
	return nil
}
