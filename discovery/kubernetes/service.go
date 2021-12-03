package kubernetes

import (
	"context"
	"fmt"

	"github.com/go-logr/zapr"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/mcorbin/cabourotte/healthcheck"
	"gopkg.in/tomb.v2"
)

const (
	defaultTargetAnnotation string = "cabourotte.mcorbin.fr/target"
)

// ServiceReconciler main service reconciler component
type ServiceReconciler struct {
	client.Client
	t                     tomb.Tomb
	Manager               ctrl.Manager
	Config                *KubernetesService
	DisableCommandsChecks bool
	Healthcheck           *healthcheck.Component
	Logger                *zap.Logger
	Controller            controller.Controller
}

// NewServiceReconciler build a service reconciler component
func NewServiceReconciler(logger *zap.Logger, healthcheck *healthcheck.Component, config *KubernetesService, disableCommandsChecks bool) (*ServiceReconciler, error) {
	kubeConfig, err := ctrl.GetConfig()
	if err != nil {
		return nil, errors.Wrapf(err, "fail to get the Kubernetes client configuration")
	}
	manager, err := ctrl.NewManager(kubeConfig,
		ctrl.Options{
			Namespace:          config.Namespace,
			MetricsBindAddress: "0",
		})
	if err != nil {
		return nil, errors.Wrapf(err, "fail to create the Kubernetes service controller manager")
	}
	reconciler := ServiceReconciler{
		Client:                manager.GetClient(),
		Manager:               manager,
		Logger:                logger,
		Config:                config,
		Healthcheck:           healthcheck,
		DisableCommandsChecks: disableCommandsChecks,
	}
	controller, err := controller.New("service-controller", manager, controller.Options{
		Reconciler: &reconciler,
	})
	ctrl.SetLogger(zapr.NewLogger(logger))
	if err != nil {
		return nil, errors.Wrapf(err, "fail to create the Kubernetes service controller")
	}
	reconciler.Controller = controller
	return &reconciler, nil
}

// Start start the service reconciler component
func (c *ServiceReconciler) Start() error {

	// Watch Services and enqueue ReplicaSet object key
	if err := c.Controller.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForObject{}); err != nil {
		c.Logger.Error(err.Error())
		return errors.Wrap(err, "fail to watch services resources")
	}

	c.t.Go(func() error {
		ctx := c.t.Context(context.TODO())
		c.Logger.Info("Starting Kubernetes service listener")
		if err := c.Manager.Start(ctx); err != nil {
			c.Logger.Error(err.Error())
			// todo: should correctly stop the daemon if it fails
			return errors.Wrap(err, "fail to start service manager")
		}
		c.Logger.Info("Stopping Kubernetes service listener")
		return nil
	})
	return nil
}

// Stop stop the service reconciler
func (c *ServiceReconciler) Stop() error {
	c.Logger.Info("stopping Kubernetes service listener")
	c.t.Kill(nil)
	err := c.t.Wait()
	if err != nil {
		return err
	}
	return nil
}

// Reconcile services healthchecks
func (c *ServiceReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	services := &corev1.ServiceList{}
	err := c.List(ctx, services, client.InNamespace(c.Config.Namespace), client.MatchingLabels(c.Config.Labels))
	if err != nil {
		return ctrl.Result{}, err
	}

	oldChecks := c.Healthcheck.SourceChecksNames(healthcheck.SourceKubernetesService)
	newChecks := make(map[string]bool)

	for _, item := range services.Items {
		terminating := item.ObjectMeta.DeletionTimestamp != nil
		serviceName := item.ObjectMeta.Name
		healthcheckType := item.ObjectMeta.Annotations[typeAnnotation]
		healthcheckLabels := item.ObjectMeta.Labels
		healthcheckTarget := item.ObjectMeta.Annotations[defaultTargetAnnotation]
		c.Logger.Info(fmt.Sprintf("Service %s detected terminating %t", serviceName, terminating))
		if !terminating && healthcheckType != "" {
			healthcheckConfig := item.ObjectMeta.Annotations[configAnnotation]
			var defaultTarget string
			if healthcheckTarget == "ip" {
				defaultTarget = item.Spec.ClusterIP
			} else {
				defaultTarget = fmt.Sprintf("%s.%s.svc", item.ObjectMeta.Name, item.ObjectMeta.Namespace)
			}
			err = addCheck(c.Healthcheck, c.Logger, newChecks, healthcheckType, healthcheckConfig, defaultTarget, healthcheck.SourceKubernetesService, healthcheckLabels, c.DisableCommandsChecks)
			if err != nil {
				c.Logger.Error(err.Error())
				return ctrl.Result{}, errors.Wrapf(err, "Fail to add healthcheck for service %s", serviceName)
			}
		}
	}
	err = c.Healthcheck.RemoveNonConfiguredHealthchecks(oldChecks, newChecks)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
