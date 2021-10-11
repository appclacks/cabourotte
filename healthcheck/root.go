package healthcheck

import (
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/pkg/errors"
	prom "github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"gopkg.in/tomb.v2"

	"cabourotte/prometheus"
)

// HealthcheckConfiguration is the interface for the healthcheck configuration
type HealthcheckConfiguration interface {
	Validate() error
}

// Healthcheck is the interface for an healthcheck
type Healthcheck interface {
	Initialize() error
	GetConfig() interface{}
	GetLabels() map[string]string
	Name() string
	Summary() string
	Execute() error
	LogDebug(message string)
	LogInfo(message string)
	OneOff() bool
	Interval() Duration
	LogError(err error, message string)
	GetTick() *time.Ticker
	SetTick(tick *time.Ticker)
	GetT() *tomb.Tomb
	Stop() error
}

// Component is the component which will manage healthchecks
type Component struct {
	Logger          *zap.Logger
	Healthchecks    map[string]Healthcheck
	resultHistogram *prom.HistogramVec
	lock            sync.RWMutex

	ChanResult chan *Result
}

// Start an healthcheck wrapper
func (c *Component) startWrapper(w Healthcheck) {
	w.LogInfo("Starting healthcheck")
	w.SetTick(time.NewTicker(time.Duration(w.Interval())))
	w.GetT().Go(func() error {
		for {
			select {
			case <-w.GetTick().C:
				start := time.Now()
				err := w.Execute()
				duration := time.Since(start)
				result := NewResult(
					w,
					duration.Seconds(),
					err)
				status := "failure"
				if result.Success {
					status = "success"
				}
				c.resultHistogram.With(prom.Labels{"name": w.Name(), "status": status}).Observe(duration.Seconds())
				c.ChanResult <- result
			case <-w.GetT().Dying():
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
		Healthchecks:    make(map[string]Healthcheck),
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
		wrapper.LogDebug("stopping healthcheck")
		err := wrapper.Stop()
		if err != nil {
			wrapper.LogError(err, "Fail to stop the healthcheck")
			return errors.Wrap(err, "Fail to stop the healthcheck component")
		}
	}
	return nil
}

// removeCheck removes an healthcheck from the component.
// The function is *not* thread-safe.
func (c *Component) removeCheck(identifier string) error {
	if existingWrapper, ok := c.Healthchecks[identifier]; ok {
		existingWrapper.LogInfo("Stopping healthcheck")
		c.resultHistogram.Delete(prom.Labels{"name": identifier, "status": "failure"})
		c.resultHistogram.Delete(prom.Labels{"name": identifier, "status": "success"})
		err := existingWrapper.Stop()
		if err != nil {
			return errors.Wrapf(err, "Fail to stop healthcheck %s", existingWrapper.Name())
		}
		delete(c.Healthchecks, identifier)
	}
	return nil
}

// AddCheck add an healthcheck to the component and starts it.
func (c *Component) AddCheck(check Healthcheck) error {
	if currentCheck, ok := c.Healthchecks[check.Name()]; ok {
		if reflect.DeepEqual(currentCheck.GetConfig(), check.GetConfig()) {
			currentCheck.LogInfo("trying to replace existing healthcheck with the same config: do nothing")
			return nil
		}
	}
	check.LogInfo("Adding healthcheck")
	err := check.Initialize()
	if err != nil {
		return errors.Wrapf(err, "Fail to initialize healthcheck %s", check.Name())
	}
	c.lock.Lock()
	defer c.lock.Unlock()

	// verifies if the healthcheck already exists, and removes it if needed.
	// Updating an healthcheck is removing the old one and adding the new one.
	err = c.removeCheck(check.Name())
	if err != nil {
		return errors.Wrapf(err, "Fail to stop existing healthcheck %s", check.Name())
	}
	c.startWrapper(check)
	c.Healthchecks[check.Name()] = check
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
	for i := range c.Healthchecks {
		wrapper := c.Healthchecks[i]
		result = append(result, wrapper)
	}
	return result
}

// GetCheck returns a check if it exists, otherwise an error.
func (c *Component) GetCheck(name string) (Healthcheck, error) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	if existingWrapper, ok := c.Healthchecks[name]; ok {
		return existingWrapper, nil
	}
	return nil, fmt.Errorf("Healthcheck %s not found", name)
}
