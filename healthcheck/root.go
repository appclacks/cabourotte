package healthcheck

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// Duration an alias for the duration type
type Duration time.Duration

// UnmarshalText unmarshal a duration
func (d *Duration) UnmarshalText(text []byte) error {
	if len(text) < 2 {
		return errors.New(fmt.Sprintf("%s is not a duration", text))
	}
	t := text[1 : len(text)-1]
	dur, err := time.ParseDuration(string(t))
	if err != nil {
		return errors.Wrapf(err, "%s is not a duration", text)
	}
	*d = Duration(dur)
	return nil
}

// UnmarshalYAML read a duration fom yaml
func (d *Duration) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var raw time.Duration
	if err := unmarshal(&raw); err != nil {
		return errors.Wrap(err, "Unable to read Cabourotte configuration")
	}
	*d = Duration(raw)
	return nil
}

// UnmarshalJSON marshal to json a duration
func (d *Duration) UnmarshalJSON(text []byte) error {
	return d.UnmarshalText(text)
}

// MarshalJSON marshal to json a duration
func (d Duration) MarshalJSON() ([]byte, error) {
	duration := time.Duration(d)
	return json.Marshal(duration.String())
}

// Result represents the result of an healthcheck
type Result struct {
	Name      string
	Success   bool
	Timestamp time.Time
	message   string
}

// Healthcheck is the face for an healthcheck
type Healthcheck interface {
	Initialize() error
	Name() string
	Start(chanResult chan *Result) error
	Stop() error
	Execute() error
	LogDebug(message string)
	LogInfo(message string)
	OneOff() bool
	LogError(err error, message string)
}

// Component is the component which will manage healthchecks
type Component struct {
	Logger       *zap.Logger
	Healthchecks map[string]Healthcheck
	lock         sync.RWMutex

	ChanResult chan *Result
}

// NewResult build a a new result for an healthcheck
func NewResult(healthcheck Healthcheck, err error) *Result {
	now := time.Now()
	result := Result{
		Name:      healthcheck.Name(),
		Timestamp: now,
	}
	if err != nil {
		result.Success = false
		result.message = err.Error()
	} else {
		result.Success = true
		result.message = "success"
	}
	return &result

}

// New creates a new Healthcheck component
func New(logger *zap.Logger, chanResult chan *Result) (*Component, error) {
	component := Component{
		Logger:       logger,
		Healthchecks: make(map[string]Healthcheck),
		ChanResult:   chanResult,
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
	c.lock.Lock()
	defer c.lock.Unlock()
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

// removeCheck removes an healthcheck from the component.
// The function is *not* thread-safe.
func (c *Component) removeCheck(identifier string) error {
	if existingCheck, ok := c.Healthchecks[identifier]; ok {
		existingCheck.LogInfo("Stopping healthcheck")
		err := existingCheck.Stop()
		if err != nil {
			return errors.Wrapf(err, "Fail to stop healthcheck %s", existingCheck.Name())
		}
		delete(c.Healthchecks, identifier)
	}
	return nil
}

// AddCheck add an healthcheck to the component and starts it.
func (c *Component) AddCheck(healthcheck Healthcheck) error {
	healthcheck.LogInfo("Adding healthcheck")
	err := healthcheck.Initialize()
	if err != nil {
		return errors.Wrapf(err, "Fail to initialize healthcheck %s", healthcheck.Name())
	}
	c.lock.Lock()
	defer c.lock.Unlock()

	// verifies if the healthcheck already exists, and removes it if needed.
	// Updating an healthcheck is removing the old one and adding the new one.
	err = c.removeCheck(healthcheck.Name())
	if err != nil {
		return errors.Wrapf(err, "Fail to stop existing healthcheck %s", healthcheck.Name())
	}
	err = healthcheck.Start(c.ChanResult)
	if err != nil {
		return errors.Wrapf(err, "Fail to start healthcheck %s", healthcheck.Name())
	}
	c.Healthchecks[healthcheck.Name()] = healthcheck
	return nil
}

// RemoveCheck Removes an healthcheck
func (c *Component) RemoveCheck(name string) error {
	c.Logger.Info(fmt.Sprintf("Removing healthcheck %s", name))
	c.lock.Lock()
	defer c.lock.Unlock()
	return c.removeCheck(name)
}

// ListChecks returns the healthchecks currently configured
func (c *Component) ListChecks() []Healthcheck {
	c.lock.RLock()
	defer c.lock.RUnlock()
	result := make([]Healthcheck, 0, len(c.Healthchecks))
	for _, value := range c.Healthchecks {
		result = append(result, value)
	}
	return result
}
