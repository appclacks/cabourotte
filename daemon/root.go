package daemon

import (
	"reflect"
	"sync"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"cabourotte/exporter"
	"cabourotte/healthcheck"
	"cabourotte/http"
	"cabourotte/memorystore"
	"cabourotte/prometheus"
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
	lock        sync.RWMutex
	ChanResult  chan *healthcheck.Result
}

// New creates and start a new daemon component
func New(logger *zap.Logger, config *Configuration) (*Component, error) {
	logger.Info("Starting the Cabourotte daemon")
	prom := prometheus.New()
	chanResult := make(chan *healthcheck.Result, config.ResultBuffer)
	checkComponent, err := healthcheck.New(logger, chanResult, prom)
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
	component := Component{
		MemoryStore: memstore,
		ChanResult:  chanResult,
		Config:      config,
		Prometheus:  prom,
		HTTP:        http,
		Logger:      logger,
		Exporter:    exporterComponent,
		Healthcheck: checkComponent,
	}
	component.ReloadHealthchecks(config)
	return &component, nil
}

// Stop stops the Cabourotte daemon
func (c *Component) Stop() error {
	c.Logger.Info("Stopping the Cabourotte daemon")
	c.lock.Lock()
	defer c.lock.Unlock()
	err := c.HTTP.Stop()
	if err != nil {
		return errors.Wrapf(err, "Fail to stop the HTTP server")
	}
	err = c.Healthcheck.Stop()
	if err != nil {
		return errors.Wrapf(err, "Fail to stop the healthcheck component")
	}
	return nil
}

func strContains(s []string, value string) bool {
	for _, v := range s {
		if v == value {
			return true
		}
	}
	return false
}

// ReloadHealthchecks reloads the healthchecks from a configuration
func (c *Component) ReloadHealthchecks(daemonConfig *Configuration) error {
	// contains the checks which were just added
	currentConfigChecks := c.Config.configChecksNames()
	newChecks := make(map[string]bool)
	for i := range daemonConfig.DNSChecks {
		config := &daemonConfig.DNSChecks[i]
		newChecks[config.Name] = true
		newCheck := healthcheck.NewDNSHealthcheck(c.Logger, config)
		err := c.Healthcheck.AddCheck(newCheck)
		if err != nil {
			return errors.Wrapf(err, "Fail to add healthcheck %s", newCheck.Name())
		}
	}
	for i := range daemonConfig.HTTPChecks {
		config := &daemonConfig.HTTPChecks[i]
		newChecks[config.Name] = true
		newCheck := healthcheck.NewHTTPHealthcheck(c.Logger, config)
		err := c.Healthcheck.AddCheck(newCheck)
		if err != nil {
			return errors.Wrapf(err, "Fail to add healthcheck %s", newCheck.Name())
		}
	}
	for i := range daemonConfig.TCPChecks {
		config := &daemonConfig.TCPChecks[i]
		newChecks[config.Name] = true
		newCheck := healthcheck.NewTCPHealthcheck(c.Logger, config)
		err := c.Healthcheck.AddCheck(newCheck)
		if err != nil {
			return errors.Wrapf(err, "Fail to add healthcheck %s", newCheck.Name())
		}
	}
	for i := range daemonConfig.TLSChecks {
		config := &daemonConfig.TLSChecks[i]
		newChecks[config.Name] = true
		newCheck := healthcheck.NewTLSHealthcheck(c.Logger, config)
		err := c.Healthcheck.AddCheck(newCheck)
		if err != nil {
			return errors.Wrapf(err, "Fail to add healthcheck %s", newCheck.Name())
		}
	}
	checks := c.Healthcheck.ListChecks()
	for i := range checks {
		check := checks[i]
		// checks added by the API should not be removed
		// on a reload if they are not in the new configuration
		if _, ok := currentConfigChecks[check.Name()]; !ok {
			continue
		}
		// if the newChecks map does not contain this healthcheck,
		// it was not added and so should be removed
		if _, ok := newChecks[check.Name()]; !ok {
			err := c.Healthcheck.RemoveCheck(check.Name())
			if err != nil {
				return errors.Wrapf(err, "Fail to remove check %s", check.Name())
			}
		}

	}
	return nil
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
	err = c.Exporter.Reload(&daemonConfig.Exporters)
	if err != nil {
		return errors.Wrapf(err, "Fail to relaod the exporters")
	}
	c.Config = daemonConfig
	return nil
}
