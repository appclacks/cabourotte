package discovery

import (
	"github.com/pkg/errors"
	"go.uber.org/zap"

	dhttp "github.com/mcorbin/cabourotte/discovery/http"
	"github.com/mcorbin/cabourotte/discovery/kubernetes"
	"github.com/mcorbin/cabourotte/healthcheck"
	"github.com/mcorbin/cabourotte/prometheus"
)

// Component contains all service discovery instances
type Component struct {
	Logger                *zap.Logger
	PodReconciler         *kubernetes.PodReconciler
	ServiceReconciler     *kubernetes.ServiceReconciler
	HealthcheckReconciler *kubernetes.HealthcheckReconciler
	HTTPDiscovery         *dhttp.HTTPDiscovery
	Prometheus            *prometheus.Prometheus
}

// New creates the main component from its configuration
func New(logger *zap.Logger, config Configuration, promComponent *prometheus.Prometheus, healthcheck *healthcheck.Component) (*Component, error) {
	component := &Component{
		Logger: logger,
	}
	if config.Kubernetes.Pod.Enabled {
		logger.Info("Building Kubernetes pod reconciler")
		podReconciler, err := kubernetes.NewPodReconciler(logger, healthcheck, &config.Kubernetes.Pod, config.Kubernetes.DisableCommandsChecks)
		if err != nil {
			return nil, errors.Wrapf(err, "Fail to create the Kubernetes pod reconciler component")
		}
		component.PodReconciler = podReconciler
	}
	if config.Kubernetes.Service.Enabled {
		logger.Info("Building Kubernetes service reconciler")
		serviceReconciler, err := kubernetes.NewServiceReconciler(logger, healthcheck, &config.Kubernetes.Service, config.Kubernetes.DisableCommandsChecks)
		if err != nil {
			return nil, errors.Wrapf(err, "Fail to create the Kubernetes pod reconciler component")
		}
		component.ServiceReconciler = serviceReconciler
	}
	if config.Kubernetes.CRD.Enabled {
		logger.Info("Building Kubernetes CRD reconciler")
		crdReconciler, err := kubernetes.NewHealthcheckReconciler(logger, healthcheck, &config.Kubernetes.CRD, config.Kubernetes.DisableCommandsChecks)
		if err != nil {
			return nil, errors.Wrapf(err, "Fail to create the Kubernetes healthcheck reconciler component")
		}
		component.HealthcheckReconciler = crdReconciler
	}
	if config.HTTP.Host != "" {
		logger.Info("Enabling HTTP discovery")
		httpDiscovery, err := dhttp.New(logger, &config.HTTP, healthcheck, promComponent)
		if err != nil {
			return nil, errors.Wrapf(err, "Fail to create the HTTP discovery component")
		}
		component.HTTPDiscovery = httpDiscovery
	}
	return component, nil
}

// Start start all discovery mechanisms
func (c *Component) Start() error {
	if c.PodReconciler != nil {
		err := c.PodReconciler.Start()
		if err != nil {
			return err
		}
	}
	if c.ServiceReconciler != nil {
		err := c.ServiceReconciler.Start()
		if err != nil {
			return err
		}
	}
	if c.HealthcheckReconciler != nil {
		err := c.HealthcheckReconciler.Start()
		if err != nil {
			return err
		}
	}
	if c.HTTPDiscovery != nil {
		err := c.HTTPDiscovery.Start()
		if err != nil {
			return err
		}
	}
	return nil
}

// Stop stop all discovery mechanisms
func (c *Component) Stop() error {
	if c.PodReconciler != nil {
		err := c.PodReconciler.Stop()
		if err != nil {
			return err
		}
	}
	if c.ServiceReconciler != nil {
		err := c.ServiceReconciler.Stop()
		if err != nil {
			return err
		}
	}
	if c.HealthcheckReconciler != nil {
		err := c.HealthcheckReconciler.Stop()
		if err != nil {
			return err
		}
	}
	if c.HTTPDiscovery != nil {
		err := c.HTTPDiscovery.Stop()
		if err != nil {
			return err
		}
	}
	return nil
}
