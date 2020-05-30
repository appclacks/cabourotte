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
	lock        sync.RWMutex
	ChanResult  chan *healthcheck.Result
}

// New creates and start a new daemon component
func New(logger *zap.Logger, config *Configuration) (*Component, error) {
	logger.Info("Starting the Cabourotte daemon")
	chanResult := make(chan *healthcheck.Result, config.ResultBuffer)
	checkComponent, err := healthcheck.New(logger, chanResult)
	if err != nil {
		return nil, errors.Wrapf(err, "Fail to create the healthcheck component")
	}
	memstore := memorystore.NewMemoryStore(logger)
	memstore.Start()
	err = checkComponent.Start()
	if err != nil {
		return nil, errors.Wrapf(err, "Fail to start the healthcheck component")
	}
	http, err := http.New(logger, memstore, &config.HTTP, checkComponent)
	if err != nil {
		return nil, errors.Wrapf(err, "Fail to create the HTTP server")
	}
	err = http.Start()
	if err != nil {
		return nil, errors.Wrapf(err, "Fail to start the HTTP server")
	}
	exporterComponent := exporter.New(logger, memstore, chanResult, &config.Exporters)
	err = exporterComponent.Start()
	if err != nil {
		return nil, errors.Wrapf(err, "Fail to start the exporter component")
	}
	component := Component{
		MemoryStore: memstore,
		ChanResult:  chanResult,
		Config:      config,
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
func (c *Component) Reload(config *Configuration) error {
	c.Logger.Info("Reloading the Cabourotte daemon")
	c.lock.Lock()
	defer c.lock.Unlock()
	// DNS healthchecks management
	var dnsChecksToRemove []string
	var dnsChecksToKeep []string
	for _, currentCheck := range c.Config.DNSChecks {
		found := false
		for _, newCheck := range config.DNSChecks {
			if currentCheck.Name == newCheck.Name {
				// check found, let's verify if the healthcheck
				// configuration is the same
				if reflect.DeepEqual(currentCheck, newCheck) {
					dnsChecksToKeep = append(dnsChecksToKeep, currentCheck.Name)
					found = true
				}
				break
			}
		}
		// check not found in the new config, it should be removed
		if !found {
			dnsChecksToRemove = append(dnsChecksToRemove, currentCheck.Name)
		}
	}
	for _, check := range dnsChecksToRemove {
		c.Healthcheck.RemoveCheck(check)
	}
	for _, newCheck := range config.DNSChecks {
		if !strContains(dnsChecksToKeep, newCheck.Name) {
			check := healthcheck.NewDNSHealthcheck(c.Logger, &newCheck)
			err := c.Healthcheck.AddCheck(check)
			if err != nil {
				return errors.Wrapf(err, "Fail to add healthcheck %s", check.Name())
			}
		}
	}
	// TCP healthchecks management, <3 golang abstractions
	var tcpChecksToRemove []string
	var tcpChecksToKeep []string
	for _, currentCheck := range c.Config.TCPChecks {
		found := false
		for _, newCheck := range config.TCPChecks {
			if currentCheck.Name == newCheck.Name {
				// check found, let's verify if the healthcheck
				// configuration is the same
				if reflect.DeepEqual(currentCheck, newCheck) {
					tcpChecksToKeep = append(tcpChecksToKeep, currentCheck.Name)
					found = true
				}
				break
			}
		}
		// check not found in the new config, it should be removed
		if !found {
			tcpChecksToRemove = append(tcpChecksToRemove, currentCheck.Name)
		}
	}
	for _, check := range tcpChecksToRemove {
		c.Healthcheck.RemoveCheck(check)
	}
	for _, newCheck := range config.TCPChecks {
		if !strContains(tcpChecksToKeep, newCheck.Name) {
			check := healthcheck.NewTCPHealthcheck(c.Logger, &newCheck)
			err := c.Healthcheck.AddCheck(check)
			if err != nil {
				return errors.Wrapf(err, "Fail to add healthcheck %s", check.Name())
			}
		}
	}
	// HTTP healthchecks management
	var httpChecksToRemove []string
	var httpChecksToKeep []string
	for _, currentCheck := range c.Config.HTTPChecks {
		found := false
		for _, newCheck := range config.HTTPChecks {
			if currentCheck.Name == newCheck.Name {
				// check found, let's verify if the healthcheck
				// configuration is the same
				if reflect.DeepEqual(currentCheck, newCheck) {
					httpChecksToKeep = append(httpChecksToKeep, currentCheck.Name)
					found = true
				}
				break
			}
		}
		// check not found in the new config, it should be removed
		if !found {
			c.Logger.Debug(fmt.Sprintf("Healthcheck %s will be removed", currentCheck.Name))
			httpChecksToRemove = append(httpChecksToRemove, currentCheck.Name)
		}
	}
	for _, check := range httpChecksToRemove {
		c.Healthcheck.RemoveCheck(check)
	}
	for _, newCheck := range config.HTTPChecks {
		if !strContains(httpChecksToKeep, newCheck.Name) {
			check := healthcheck.NewHTTPHealthcheck(c.Logger, &newCheck)
			err := c.Healthcheck.AddCheck(check)
			if err != nil {
				return errors.Wrapf(err, "Fail to add healthcheck %s", check.Name())
			}
		}
	}
	// compare the server config to see if we need to recreate it
	if !reflect.DeepEqual(c.Config.HTTP, config) {
		err := c.HTTP.Stop()
		if err != nil {
			return errors.Wrapf(err, "Fail to stop the HTTP server")
		}
		http, err := http.New(c.Logger, c.MemoryStore, &config.HTTP, c.Healthcheck)
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
	exporterComponent := exporter.New(c.Logger, c.MemoryStore, c.ChanResult, &config.Exporters)
	err = exporterComponent.Start()
	if err != nil {
		return errors.Wrapf(err, "Fail to start the exporter component")
	}
	c.Exporter = exporterComponent
	return nil
}
