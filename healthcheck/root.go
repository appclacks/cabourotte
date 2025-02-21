package healthcheck

import (
	"context"
	"fmt"
	"math/rand"
	"reflect"
	"sort"
	"sync"
	"time"

	"github.com/pkg/errors"
	prom "github.com/prometheus/client_golang/prometheus"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.uber.org/zap"

	"github.com/appclacks/cabourotte/prometheus"
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
	Execute(ctx *context.Context) error
	LogDebug(message string)
	LogInfo(message string)
	Base() Base
	SetSource(source string)
	LogError(err error, message string)
}

// Component is the component which will manage healthchecks
type Component struct {
	Logger             *zap.Logger
	Healthchecks       map[string]*Wrapper
	resultHistogram    *prom.HistogramVec
	resultCounter      *prom.CounterVec
	lock               sync.RWMutex
	healthchecksLabels []string

	ChanResult chan *Result
}

// Start an healthcheck wrapper
func (c *Component) startWrapper(w *Wrapper) {
	tracer := otel.Tracer("healthcheck")
	w.healthcheck.LogInfo("Starting healthcheck")
	w.Tick = time.NewTicker(time.Duration(w.healthcheck.Base().Interval))
	w.t.Go(func() error {
		wait := rand.Intn(4000)
		time.Sleep(time.Duration(wait) * time.Millisecond)
		for {
			ctx, span := tracer.Start(context.Background(), "healthcheck.periodic")
			span.SetAttributes(attribute.String("cabourotte.healthcheck.name", w.healthcheck.Base().Name))
			start := time.Now()
			err := w.healthcheck.Execute(&ctx)
			duration := time.Since(start)
			result := NewResult(
				w.healthcheck,
				duration.Milliseconds(),
				map[string]string{},
				err)
			if ctx.Value("labels") != nil {
				result.MessageLabels = ctx.Value("labels").(map[string]string)
			}
			status := "failure"
			if result.Success {
				status = "success"
				span.SetStatus(codes.Ok, "healthcheck successful")
			} else {
				span.RecordError(err)
				span.SetStatus(codes.Error, "healthcheck failure")
			}
			span.SetAttributes(attribute.String("cabourotte.healthcheck.status", status))
			for k, v := range w.healthcheck.Base().Labels {
				span.SetAttributes(attribute.String(fmt.Sprintf("cabourotte.healthcheck.label.%s", k), v))
			}
			span.End()
			histoLabels := map[string]string{
				"name": w.healthcheck.Base().Name,
			}
			for _, k := range c.healthchecksLabels {
				histoLabels[k] = result.Labels[k]
			}
			c.resultHistogram.With(prom.Labels(histoLabels)).Observe(duration.Seconds())
			counterLabels := map[string]string{
				"name":   w.healthcheck.Base().Name,
				"status": status,
			}
			for _, k := range c.healthchecksLabels {
				counterLabels[k] = result.Labels[k]
			}
			c.resultCounter.With(prom.Labels(counterLabels)).Inc()
			c.ChanResult <- result
			select {
			case <-w.Tick.C:
				continue
			case <-w.t.Dying():
				return nil
			}
		}
	})
}

// New creates a new Healthcheck component
func New(logger *zap.Logger, chanResult chan *Result, promComponent *prometheus.Prometheus, healthchecksLabels []string) (*Component, error) {
	buckets := []float64{
		0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 0.75, 1,
		2.5, 5, 7.5, 10}
	histoLabels := []string{"name"}
	histoLabels = append(histoLabels, healthchecksLabels...)
	histo := prom.NewHistogramVec(prom.HistogramOpts{
		Name:    "healthcheck_duration_seconds",
		Help:    "Time to execute a healthcheck.",
		Buckets: buckets,
	},
		histoLabels,
	)
	counterLabels := []string{"name", "status"}
	counterLabels = append(counterLabels, healthchecksLabels...)
	counter := prom.NewCounterVec(
		prom.CounterOpts{
			Name: "healthcheck_total",
			Help: "Count the number of healthchecks executions.",
		},
		counterLabels)

	err := promComponent.Register(histo)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to register the healthcheck results Prometheus histogram")
	}
	err = promComponent.Register(counter)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to register the healthcheck results Prometheus counter")
	}
	component := Component{
		resultCounter:      counter,
		resultHistogram:    histo,
		Logger:             logger,
		Healthchecks:       make(map[string]*Wrapper),
		ChanResult:         chanResult,
		healthchecksLabels: healthchecksLabels,
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
	c.Logger.Info("All healthchecks stopped")
	return nil
}

// removeCheck removes an healthcheck from the component.
// The function is *not* thread-safe.
func (c *Component) removeCheck(identifier string) error {
	if existingWrapper, ok := c.Healthchecks[identifier]; ok {
		existingWrapper.healthcheck.LogInfo("Stopping healthcheck")
		c.resultHistogram.DeletePartialMatch(prom.Labels{"name": identifier})
		c.resultCounter.DeletePartialMatch(prom.Labels{"name": identifier})
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

// ListChecks returns the healthchecks currently configured, sorted by name
func (c *Component) ListChecks() []Healthcheck {
	c.lock.RLock()
	defer c.lock.RUnlock()
	result := make([]Healthcheck, 0, len(c.Healthchecks))
	for i := range c.Healthchecks {
		wrapper := c.Healthchecks[i]
		result = append(result, wrapper.healthcheck)
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Base().Name < result[j].Base().Name
	})
	return result
}

// GetCheck returns a check if it exists, otherwise an error.
func (c *Component) GetCheck(name string) Healthcheck {
	c.lock.RLock()
	defer c.lock.RUnlock()
	if existingWrapper, ok := c.Healthchecks[name]; ok {
		return existingWrapper.healthcheck
	}
	return nil
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

// MergeLabels merge labels from a base and a map of string
func MergeLabels(base *Base, new map[string]string) {
	if new != nil && base.Labels != nil {
		for k, v := range new {
			base.Labels[k] = v
		}
	}
	if new != nil && base.Labels == nil {
		base.Labels = new
	}

}

func (c *Component) ReloadForSource(
	source string,
	commonLabels map[string]string,
	command []CommandHealthcheckConfiguration,
	dns []DNSHealthcheckConfiguration,
	tcp []TCPHealthcheckConfiguration,
	http []HTTPHealthcheckConfiguration,
	tls []TLSHealthcheckConfiguration) error {

	oldChecks := c.SourceChecksNames(source)
	newChecks := make(map[string]bool)
	for i := range command {
		config := &command[i]
		MergeLabels(&config.Base, commonLabels)
		config.Base.Source = source
		newChecks[config.Base.Name] = true
		err := config.Validate()
		if err != nil {
			return err
		}
		newCheck := NewCommandHealthcheck(c.Logger, config)
		err = c.AddCheck(newCheck)
		if err != nil {
			return errors.Wrapf(err, "Fail to add healthcheck %s", newCheck.Base().Name)
		}
	}
	for i := range dns {
		config := &dns[i]
		MergeLabels(&config.Base, commonLabels)
		config.Base.Source = source
		newChecks[config.Base.Name] = true
		err := config.Validate()
		if err != nil {
			return err
		}
		newCheck := NewDNSHealthcheck(c.Logger, config)
		err = c.AddCheck(newCheck)
		if err != nil {
			return errors.Wrapf(err, "Fail to add healthcheck %s", newCheck.Base().Name)
		}
	}
	for i := range http {
		config := &http[i]
		MergeLabels(&config.Base, commonLabels)
		config.Base.Source = source
		newChecks[config.Base.Name] = true
		err := config.Validate()
		if err != nil {
			return err
		}
		newCheck := NewHTTPHealthcheck(c.Logger, config)
		err = c.AddCheck(newCheck)
		if err != nil {
			return errors.Wrapf(err, "Fail to add healthcheck %s", newCheck.Base().Name)
		}
	}
	for i := range tcp {
		config := &tcp[i]
		MergeLabels(&config.Base, commonLabels)
		config.Base.Source = source
		newChecks[config.Base.Name] = true
		err := config.Validate()
		if err != nil {
			return err
		}
		newCheck := NewTCPHealthcheck(c.Logger, config)
		err = c.AddCheck(newCheck)
		if err != nil {
			return errors.Wrapf(err, "Fail to add healthcheck %s", newCheck.Base().Name)
		}
	}
	for i := range tls {
		config := &tls[i]
		MergeLabels(&config.Base, commonLabels)
		config.Base.Source = source
		newChecks[config.Base.Name] = true
		err := config.Validate()
		if err != nil {
			return err
		}
		newCheck := NewTLSHealthcheck(c.Logger, config)
		err = c.AddCheck(newCheck)
		if err != nil {
			return errors.Wrapf(err, "Fail to add healthcheck %s", newCheck.Base().Name)
		}
	}
	return c.RemoveNonConfiguredHealthchecks(oldChecks, newChecks)
}
