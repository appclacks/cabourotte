package healthcheck

import (
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
	LogError(err error, message string)
}

// Component is the component which will manage healthchecks
type Component struct {
	Logger       *zap.Logger
	Healthchecks map[string]Healthcheck
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

func (c *Component) AddCheck(Healthcheck *Healthcheck) error {
	return nil
}
