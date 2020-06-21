package healthcheck

import (
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	prom "github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"gopkg.in/tomb.v2"

	"cabourotte/prometheus"
)

// Result represents the result of an healthcheck
type Result struct {
	Name      string    `json:"name"`
	Success   bool      `json:"success"`
	Timestamp time.Time `json:"timestamp"`
	Message   string    `json:"message"`
}

// Wrapper Wrap an healthcheck
type Wrapper struct {
	healthcheck Healthcheck
	Tick        *time.Ticker
	t           tomb.Tomb
}

// NewWrapper creates a new wrapper struct
func NewWrapper(healthcheck Healthcheck) *Wrapper {
	return &Wrapper{
		healthcheck: healthcheck,
	}
}

// Stop an Healthcheck wrapper
func (w *Wrapper) Stop() error {
	w.Tick.Stop()
	w.t.Kill(nil)
	w.t.Wait()
	return nil

}

// Healthcheck is the face for an healthcheck
type Healthcheck interface {
	Initialize() error
	Name() string
	Execute() error
	LogDebug(message string)
	LogInfo(message string)
	OneOff() bool
	Interval() Duration
	LogError(err error, message string)
}

// Component is the component which will manage healthchecks
type Component struct {
	Logger          *zap.Logger
	Healthchecks    map[string]*Wrapper
	resultCounter   *prom.CounterVec
	resultHistogram *prom.HistogramVec
	lock            sync.RWMutex

	ChanResult chan *Result
}

// Start an healthcheck wrapper
func (c *Component) startWrapper(w *Wrapper) {
	w.healthcheck.LogInfo("Starting healthcheck")
	w.Tick = time.NewTicker(time.Duration(w.healthcheck.Interval()))
	w.t.Go(func() error {
		for {
			select {
			case <-w.Tick.C:
				start := time.Now()
				err := w.healthcheck.Execute()
				duration := time.Since(start)
				result := NewResult(w.healthcheck, err)
				status := "failure"
				if result.Success {
					status = "success"
				}
				c.resultCounter.With(prom.Labels{"name": w.healthcheck.Name(), "status": status}).Inc()
				c.resultHistogram.With(prom.Labels{"name": w.healthcheck.Name(), "status": status}).Observe(duration.Seconds())
				c.ChanResult <- result
			case <-w.t.Dying():
				return nil
			}
		}
	})
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
		result.Message = err.Error()
	} else {
		result.Success = true
		result.Message = "success"
	}
	return &result
}

// New creates a new Healthcheck component
func New(logger *zap.Logger, chanResult chan *Result, promComponent *prometheus.Prometheus) (*Component, error) {
	counter := prom.NewCounterVec(
		prom.CounterOpts{
			Name: "healthcheck_result_total",
			Help: "Count the healthchecks of success or failures for healthchchecks.",
		},
		[]string{"name", "status"},
	)
	buckets := []float64{
		0.05, 0.1, 0.2, 0.4, 0.8, 1,
		1.5, 2, 3, 5}
	histo := prom.NewHistogramVec(prom.HistogramOpts{
		Name:    "healthcheck_duration_seconds",
		Help:    "Time to execute a healthcheck healthcheck.",
		Buckets: buckets,
	},
		[]string{"name", "status"},
	)
	err := promComponent.Register(counter)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to register the healthcheck result Prometheus counter")
	}
	err = promComponent.Register(histo)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to register the healthcheck result Prometheus histogram")
	}
	component := Component{
		resultCounter:   counter,
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
	for _, wrapper := range c.Healthchecks {
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
		err := existingWrapper.Stop()
		if err != nil {
			return errors.Wrapf(err, "Fail to stop healthcheck %s", existingWrapper.healthcheck.Name())
		}
		delete(c.Healthchecks, identifier)
	}
	return nil
}

// AddCheck add an healthcheck to the component and starts it.
func (c *Component) AddCheck(check Healthcheck) error {
	wrapper := NewWrapper(check)
	wrapper.healthcheck.LogInfo("Adding healthcheck")
	err := wrapper.healthcheck.Initialize()
	if err != nil {
		return errors.Wrapf(err, "Fail to initialize healthcheck %s", wrapper.healthcheck.Name())
	}
	c.lock.Lock()
	defer c.lock.Unlock()

	// verifies if the healthcheck already exists, and removes it if needed.
	// Updating an healthcheck is removing the old one and adding the new one.
	err = c.removeCheck(wrapper.healthcheck.Name())
	if err != nil {
		return errors.Wrapf(err, "Fail to stop existing healthcheck %s", wrapper.healthcheck.Name())
	}
	c.startWrapper(wrapper)
	c.Healthchecks[wrapper.healthcheck.Name()] = wrapper
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
	for _, wrapper := range c.Healthchecks {
		result = append(result, wrapper.healthcheck)
	}
	return result
}
