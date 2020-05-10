package healthcheck

import (
	"sync"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// Healthcheck is the face for an healthcheck
type Healthcheck interface {
	Initialize() error
	Identifier() string
	Start() error
	Stop() error
	Execute() error
	LogDebug(message string)
	LogInfo(message string)
	LogError(err error, message string)
}

// Component is the component which will manage healthchecks
type Component struct {
	Logger       *zap.Logger
	Healthchecks map[string]Healthcheck
	lock         sync.RWMutex
}

// New creates a new Healthcheck component
func New(logger *zap.Logger) (*Component, error) {
	component := Component{
		Logger:       logger,
		Healthchecks: make(map[string]Healthcheck),
	}

	return &component, nil

}

// Start start the healthcheck component
func (c *Component) Start() error {
	c.Logger.Info("Starting the healthcheck component")
	// nothing to do
	return nil
}

// Stop stop the healthcheck component, stopping all healthchecks being executed.
func (c *Component) Stop() error {
	c.Logger.Info("Stopping the healthcheck component")
	for _, healthcheck := range c.Healthchecks {
		healthcheck.LogDebug("stopping healthcheck")
		err := healthcheck.Stop()
		if err != nil {
			healthcheck.LogError(err, "Fail to stop the healthcheck")
			return errors.Wrap(err, "Fail to stop the healthcheck component")
		}
	}
	return nil
}

func (c *Component) removeCheck(identifier string) error {
	if existingCheck, ok := c.Healthchecks[identifier]; ok {
		existingCheck.LogInfo("Stopping healthcheck")
		err := existingCheck.Stop()
		if err != nil {
			return errors.Wrapf(err, "Fail to stop healthcheck %s", existingCheck.Identifier())
		}
		delete(c.Healthchecks, identifier)
	}
	return nil
}

// AddCheck add an healthcheck to the component and starts it.
func (c *Component) AddCheck(healthcheck Healthcheck) error {
	err := healthcheck.Initialize()
	if err != nil {
		return errors.Wrapf(err, "Fail to initialize healthcheck %s", healthcheck.Identifier())
	}
	c.lock.Lock()
	defer c.lock.Unlock()

	// verifies if the healthcheck already exists, and removes it if needed.
	// Updating an healthcheck is removing the old one and adding the new one.
	err = c.removeCheck(healthcheck.Identifier())
	if err != nil {
		return errors.Wrapf(err, "Fail to stop existing healthcheck %s", healthcheck.Identifier())
	}
	err = healthcheck.Start()
	if err != nil {
		return errors.Wrapf(err, "Fail to start healthcheck %s", healthcheck.Identifier())
	}
	c.Healthchecks[healthcheck.Identifier()] = healthcheck
	return nil
}

// RemoveCheck Removes an healthchec
func (c *Component) RemoveCheck(identifier string) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.removeCheck(identifier)
}
