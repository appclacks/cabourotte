package healthcheck

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/pkg/errors"
	prom "github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"

	"github.com/mcorbin/cabourotte/prometheus"
)

// HealthcheckConfiguration is the interface for the healthcheck configuration
type HealthcheckConfiguration interface {
	Validate() error
}

// Healthcheck is the interface for an healthcheck
type Healthcheck interface {
	Initialize() error
	GetConfig() interface{}
	Summary() string
	Execute() error
	LogDebug(message string)
	LogInfo(message string)
	Base() Base
	SetSource(source string)
	LogError(err error, message string)
}

// Component is the component which will manage healthchecks
type Component struct {
	Logger          *zap.Logger
	Healthchecks    map[string]*Wrapper
	resultHistogram *prom.HistogramVec
	lock            sync.RWMutex

	ChanResult chan *Result
}

// Start an healthcheck wrapper
func (c *Component) startWrapper(w *Wrapper) {
	w.healthcheck.LogInfo("Starting healthcheck")
	w.Tick = time.NewTicker(time.Duration(w.healthcheck.Base().Interval))
	w.t.Go(func() error {
		for {
			select {
			case <-w.Tick.C:
				start := time.Now()
				err := w.healthcheck.Execute()
				duration := time.Since(start)
				result := NewResult(
					w.healthcheck,
					duration.Seconds(),
					err)
				status := "failure"
				if result.Success {
					status = "success"
				}
				c.resultHistogram.With(prom.Labels{"name": w.healthcheck.Base().Name, "status": status}).Observe(duration.Seconds())
				c.ChanResult <- result
			case <-w.t.Dying():
				return nil
			}
		}
	})
}

// New creates a new Healthcheck component
func New(logger *zap.Logger, chanResult chan *Result, promComponent *prometheus.Prometheus) (*Component, error) {
	buckets := []float64{
		0.05, 0.1, 0.2, 0.4, 0.8, 1,
		1.5, 2, 3, 5}
	histo := prom.NewHistogramVec(prom.HistogramOpts{
		Name:    "healthcheck_duration_seconds",
		Help:    "Time to execute a healthcheck.",
		Buckets: buckets,
	},
		[]string{"name", "status"},
	)
	err := promComponent.Register(histo)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to register the healthcheck result Prometheus histogram")
	}
	component := Component{
		resultHistogram: histo,
		Logger:          logger,
		Healthchecks:    make(map[string]*Wrapper),
		ChanResult:      chanResult,
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
	for i := range c.Healthchecks {
		wrapper := c.Healthchecks[i]
		wrapper.healthcheck.LogDebug("stopping healthcheck")
		err := wrapper.Stop()
		if err != nil {
			wrapper.healthcheck.LogError(err, "Fail to stop the healthcheck")
			return errors.Wrap(err, "Fail to stop the healthcheck component")
		}
	}
	return nil
}

// removeCheck removes an healthcheck from the component.
// The function is *not* thread-safe.
func (c *Component) removeCheck(identifier string) error {
	if existingWrapper, ok := c.Healthchecks[identifier]; ok {
		existingWrapper.healthcheck.LogInfo("Stopping healthcheck")
		c.resultHistogram.Delete(prom.Labels{"name": identifier, "status": "failure"})
		c.resultHistogram.Delete(prom.Labels{"name": identifier, "status": "success"})
		err := existingWrapper.Stop()
		if err != nil {
			return errors.Wrapf(err, "Fail to stop healthcheck %s", existingWrapper.healthcheck.Base().Name)
		}
		delete(c.Healthchecks, identifier)
		existingWrapper.healthcheck.LogInfo("Healthcheck stopped")
	}
	return nil
}

// AddCheck add an healthcheck to the component and starts it.
func (c *Component) AddCheck(check Healthcheck) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	if currentCheck, ok := c.Healthchecks[check.Base().Name]; ok {
		if reflect.DeepEqual(currentCheck.healthcheck.GetConfig(), check.GetConfig()) {
			currentCheck.healthcheck.LogDebug("trying to replace existing healthcheck with the same config: do nothing")
			return nil
		}
	}
	wrapper := NewWrapper(check)
	wrapper.healthcheck.LogInfo("Adding healthcheck")
	err := wrapper.healthcheck.Initialize()
	if err != nil {
		return errors.Wrapf(err, "Fail to initialize healthcheck %s", wrapper.healthcheck.Base().Name)
	}

	// verifies if the healthcheck already exists, and removes it if needed.
	// Updating an healthcheck is removing the old one and adding the new one.
	err = c.removeCheck(wrapper.healthcheck.Base().Name)
	if err != nil {
		return errors.Wrapf(err, "Fail to stop existing healthcheck %s", wrapper.healthcheck.Base().Name)
	}
	c.startWrapper(wrapper)
	c.Healthchecks[wrapper.healthcheck.Base().Name] = wrapper
	return nil
}

// RemoveCheck Removes an healthcheck
func (c *Component) RemoveCheck(name string) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.Logger.Info(fmt.Sprintf("Removing healthcheck %s", name))
	return c.removeCheck(name)
}

// ListChecks returns the healthchecks currently configured
func (c *Component) ListChecks() []Healthcheck {
	c.lock.RLock()
	defer c.lock.RUnlock()
	result := make([]Healthcheck, 0, len(c.Healthchecks))
	for i := range c.Healthchecks {
		wrapper := c.Healthchecks[i]
		result = append(result, wrapper.healthcheck)
	}
	return result
}

// GetCheck returns a check if it exists, otherwise an error.
func (c *Component) GetCheck(name string) (Healthcheck, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	if existingWrapper, ok := c.Healthchecks[name]; ok {
		return existingWrapper.healthcheck, nil
	}
	return nil, fmt.Errorf("Healthcheck %s not found", name)
}

// RemoveNonConfiguredHealthchecks takes two list of healthchecks. Delete from the
// healthcheck component the checks which exist in the first list but not in the
// second one
func (c *Component) RemoveNonConfiguredHealthchecks(oldChecks map[string]bool, newChecks map[string]bool) error {
	for check := range oldChecks {
		// checks which are present in both old and current config
		// should be kept
		if _, ok := newChecks[check]; ok {
			continue
		}
		// the rest should be deleted
		if _, ok := newChecks[check]; !ok {
			err := c.RemoveCheck(check)
			if err != nil {
				return errors.Wrapf(err, "Fail to remove check %s", check)
			}
		}

	}
	return nil
}
