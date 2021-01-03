package exporter

import (
	"fmt"
	"reflect"
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
			return errors.Wrapf(err, "fail to start the exporter %s", exporter.Name())
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
					start := time.Now()
					err := exporter.Push(message)
					duration := time.Since(start)
					status := "success"
					name := exporter.Name()
					if err != nil {
						c.Logger.Error(fmt.Sprintf("Failed to push healthchecks result for exporter %s: %s", name, err.Error()))
						status = "failure"
					}
					c.exporterHistogram.With(prom.Labels{"name": name, "status": status}).Observe(duration.Seconds())
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

func (c *Component) reload(config interface{}, configName string, exporterType string) error {
	recreate := true
	if exporter, ok := c.Exporters[configName]; ok {
		if reflect.DeepEqual(exporter.GetConfig(), config) {
			recreate = false
		} else {
			c.Logger.Info(fmt.Sprintf("Recreating exporter %s", configName))
			err := exporter.Stop()
			if err != nil {
				return errors.Wrapf(err, "fail to create the http exporter %s", configName)
			}
		}
	}
	if recreate {
		var exporter Exporter
		var err error
		if exporterType == "http" {
			conf := config.(*HTTPConfiguration)
			exporter, err = NewHTTPExporter(c.Logger, conf)
		}
		if err != nil {
			return errors.Wrapf(err, "fail to create the http exporter %s", configName)
		}
		err = exporter.Start()
		if err != nil {
			return errors.Wrapf(err, "fail to create the http exporter %s", configName)
		}
		c.Exporters[configName] = exporter
	}
	return nil
}

// Reload reloads the Exporter component.
func (c *Component) Reload(config *Configuration) error {
	c.lock.Lock()
	defer c.lock.Unlock()
	for i := range config.HTTP {
		httpConfig := config.HTTP[i]
		err := c.reload(&httpConfig, httpConfig.Name, "http")
		if err != nil {
			return err
		}
	}
	return nil
}

// Stop the exporters
func (c *Component) Stop() error {
	c.lock.Lock()
	defer c.lock.Unlock()
	c.t.Kill(nil)
	c.t.Wait()
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
