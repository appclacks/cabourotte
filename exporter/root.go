package exporter

import (
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	prom "github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
	"gopkg.in/tomb.v2"

	"cabourotte/healthcheck"
	"cabourotte/memorystore"
	"cabourotte/prometheus"
)

// Exporter the exporter interface
type Exporter interface {
	Start() error
	Stop() error
	Reconnect() error
	IsStarted() bool
	Name() string
	GetConfig() interface{}
	Push(*healthcheck.Result) error
}

// Component the exporter component
type Component struct {
	Logger            *zap.Logger
	Config            *Configuration
	ChanResult        chan *healthcheck.Result
	Exporters         map[string]Exporter
	MemoryStore       *memorystore.MemoryStore
	exporterHistogram *prom.HistogramVec
	chanResultGauge   *prom.GaugeVec
	prometheus        *prometheus.Prometheus
	gaugeTick         *time.Ticker
	lock              sync.RWMutex

	t tomb.Tomb
}

// New creates a new exporter component
func New(logger *zap.Logger, store *memorystore.MemoryStore, chanResult chan *healthcheck.Result, promComponent *prometheus.Prometheus, config *Configuration) (*Component, error) {
	exporters := make(map[string]Exporter)
	for i := range config.HTTP {
		httpConfig := config.HTTP[i]
		exporter, err := NewHTTPExporter(logger, &httpConfig)
		if err != nil {
			return nil, errors.Wrapf(err, "fail to create the http exporter")
		}
		exporters[httpConfig.Name] = exporter
	}
	for i := range config.Riemann {
		riemannConfig := config.Riemann[i]
		exporter, err := NewRiemannExporter(logger, &riemannConfig)
		if err != nil {
			return nil, errors.Wrapf(err, "fail to create the http exporter")
		}
		exporters[riemannConfig.Name] = exporter
	}
	buckets := []float64{
		0.05, 0.1, 0.2, 0.4, 0.8, 1,
		1.5, 2, 3, 5}
	histo := prom.NewHistogramVec(prom.HistogramOpts{
		Name:    "exporter_duration_seconds",
		Help:    "Time to push to an exporter.",
		Buckets: buckets,
	},
		[]string{"name", "status"})
	gauge := prom.NewGaugeVec(prom.GaugeOpts{
		Name: "result_chan_size",
		Help: "Size of the result channel.",
	}, []string{})
	err := promComponent.Register(histo)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to register the exporter Prometheus histogram")
	}
	err = promComponent.Register(gauge)
	if err != nil {
		return nil, errors.Wrapf(err, "fail to register the chan result Prometheus gauge")
	}
	return &Component{
		exporterHistogram: histo,
		chanResultGauge:   gauge,
		MemoryStore:       store,
		Logger:            logger,
		Config:            config,
		ChanResult:        chanResult,
		Exporters:         exporters,
		prometheus:        promComponent,
		gaugeTick:         time.NewTicker(time.Duration(time.Second * 10)),
	}, nil
}

// Start starts the exporter component
func (c *Component) Start() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.Logger.Info("Starting the exporters")
	for _, exporter := range c.Exporters {
		err := exporter.Start()
		if err != nil {
			// do not return error on purpose, clients should be able to reconnect
			c.Logger.Error(fmt.Sprintf("fail to create the exporter %s: %s", exporter.Name(), err.Error()))
		}
	}
	c.t.Go(func() error {
		for {
			select {
			case <-c.gaugeTick.C:
				c.chanResultGauge.WithLabelValues().Set(float64(len(c.ChanResult)))
			case <-c.t.Dying():
				return nil
			}
		}
	})
	c.t.Go(func() error {
		for {
			select {
			case message := <-c.ChanResult:
				c.MemoryStore.Add(message)
				if message.Success {
					c.Logger.Info("Healthcheck successful",
						zap.String("name", message.Name),
						zap.Reflect("labels", message.Labels),
						zap.Int64("healthcheck-timestamp", message.HealthcheckTimestamp),
					)
				} else {
					c.Logger.Error("healthcheck failed",
						zap.String("name", message.Name),
						zap.Reflect("labels", message.Labels),
						zap.String("cause", message.Message),
						zap.Int64("healthcheck-timestamp", message.HealthcheckTimestamp),
					)
				}
				c.lock.Lock()
				for k := range c.Exporters {
					exporter := c.Exporters[k]
					if exporter.IsStarted() {
						start := time.Now()
						err := exporter.Push(message)
						duration := time.Since(start)
						status := "success"
						name := exporter.Name()
						if err != nil {
							c.Logger.Error(fmt.Sprintf("Failed to push healthchecks result for exporter %s: %s", name, err.Error()))
							status = "failure"
							err := exporter.Stop()
							if err != nil {
								// do not return error
								// on purpose
								c.Logger.Error(fmt.Sprintf("Fail to close the exporter %s: %s", name, err.Error()))
							}
						}
						c.exporterHistogram.With(prom.Labels{"name": name, "status": status}).Observe(duration.Seconds())
					}
					if !exporter.IsStarted() {
						err := exporter.Reconnect()
						if err != nil {
							// do not return error
							// on purpose
							c.Logger.Error(fmt.Sprintf("fail to reconnect the exporter %s: %s", exporter.Name(), err.Error()))
						}
					}
				}
				c.lock.Unlock()
			case <-c.t.Dying():
				return nil
			}
		}
	})
	// nothing to do
	return nil
}

// Stop the exporters
func (c *Component) Stop() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.t.Kill(nil)
	err := c.t.Wait()
	if err != nil {
		return err
	}
	c.prometheus.Unregister(c.chanResultGauge)
	c.prometheus.Unregister(c.exporterHistogram)
	for k := range c.Exporters {
		e := c.Exporters[k]
		err := e.Stop()
		if err != nil {
			return errors.Wrapf(err, "Fail to stop an exporter")
		}
	}
	return nil
}
