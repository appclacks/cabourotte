package daemon

import (
	"fmt"
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
	// start all checks
	for _, checkConfig := range config.DNSChecks {
		check := healthcheck.NewDNSHealthcheck(logger, &checkConfig)
		err := checkComponent.AddCheck(check)
		if err != nil {
			return nil, errors.Wrapf(err, "Fail to add healthcheck %s", check.Name())
		}
	}

	for _, checkConfig := range config.TCPChecks {
		check := healthcheck.NewTCPHealthcheck(logger, &checkConfig)
		err := checkComponent.AddCheck(check)
		if err != nil {
			return nil, errors.Wrapf(err, "Fail to add healthcheck %s", check.Name())
		}
	}

	for _, checkConfig := range config.HTTPChecks {
		check := healthcheck.NewHTTPHealthcheck(logger, &checkConfig)
		err := checkComponent.AddCheck(check)
		if err != nil {
			return nil, errors.Wrapf(err, "Fail to add healthcheck %s", check.Name())
		}
	}
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

// Reload reloads the Cabourotte daemon. This function will remove or keep
// existing healthchecks depending of the new configuration. New checks will be added.
// The HTTP server will also be reloaded if its configuration has changed.
func (c *Component) Reload(daemonConfig *Configuration) error {
	c.Logger.Info("Reloading the Cabourotte daemon")
	c.lock.Lock()
	defer c.lock.Unlock()
	var checksToRemove []string
	var checksToKeep []string
	// the new configurations
	var configurations []healthcheck.HealthcheckConfiguration
	for i := range daemonConfig.DNSChecks {
		configurations = append(configurations, &daemonConfig.DNSChecks[i])
	}
	for i := range daemonConfig.HTTPChecks {
		configurations = append(configurations, &daemonConfig.HTTPChecks[i])
	}

	for i := range daemonConfig.TCPChecks {
		configurations = append(configurations, &daemonConfig.TCPChecks[i])
	}

	for _, currentCheck := range c.Healthcheck.ListChecks() {
		found := false
		// iterate on the new Configurations
		for _, config := range configurations {
			if currentCheck.Name() == config.GetName() {
				// check found in the new config
				// let's verify if the healthcheck
				// configuration is the same
				if reflect.DeepEqual(currentCheck.GetConfig(), config) {
					// if it's equal, we want to keep it and not modify it
					checksToKeep = append(checksToKeep, currentCheck.Name())
					found = true
				}
				break
			}
		}
		// check not found in the new config, it should be removed
		if !found {
			checksToRemove = append(checksToRemove, currentCheck.Name())
		}
	}
	// remove checks which do not exist anymore
	for _, check := range checksToRemove {
		c.Healthcheck.RemoveCheck(check)
	}
	// Iterate again on the new configurations
	for _, config := range configurations {
		// If the configuration is a new one, or if an healthcheck was updated,
		// we create an healthcheck from the config.
		if !strContains(checksToKeep, config.GetName()) {
			var newCheck healthcheck.Healthcheck
			switch t := config.(type) {
			case *healthcheck.HTTPHealthcheckConfiguration:
				checkConfig, ok := config.(*healthcheck.HTTPHealthcheckConfiguration)
				if !ok {
					return fmt.Errorf("Fail to create the HTTP healthcheck configuration for check %s", config.GetName())
				}
				newCheck = healthcheck.NewHTTPHealthcheck(c.Logger, checkConfig)

			case *healthcheck.TCPHealthcheckConfiguration:
				checkConfig, ok := config.(*healthcheck.TCPHealthcheckConfiguration)
				if !ok {
					return fmt.Errorf("Fail to create the TCP healthcheck configuration for check %s", config.GetName())
				}
				newCheck = healthcheck.NewTCPHealthcheck(c.Logger, checkConfig)
			case *healthcheck.DNSHealthcheckConfiguration:
				checkConfig, ok := config.(*healthcheck.DNSHealthcheckConfiguration)
				if !ok {
					return fmt.Errorf("Fail to create the DNS healthcheck configuration for check %s", config.GetName())
				}
				newCheck = healthcheck.NewDNSHealthcheck(c.Logger, checkConfig)
			default:

				return fmt.Errorf("Invalid healthcheck type during reload: %v", t)
			}
			err := c.Healthcheck.AddCheck(newCheck)
			if err != nil {
				return errors.Wrapf(err, "Fail to add healthcheck %s", newCheck.Name())
			}
		}
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
	// Stop all exporters, recreate the component
	err := c.Exporter.Stop()
	if err != nil {
		return errors.Wrapf(err, "Fail to stop the exporter component")
	}
	exporterComponent, err := exporter.New(c.Logger, c.MemoryStore, c.ChanResult, c.Prometheus, &daemonConfig.Exporters)
	if err != nil {
		return errors.Wrapf(err, "Fail to create the exporter component")
	}
	err = exporterComponent.Start()
	if err != nil {
		return errors.Wrapf(err, "Fail to start the exporter component")
	}
	c.Exporter = exporterComponent
	return nil
}
