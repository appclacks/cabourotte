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
	typeAnnotation   string = "cabourotte.mcorbin.fr/healthcheck-type"
	configAnnotation string = "cabourotte.mcorbin.fr/healthcheck-configuration"
)

// PodReconciler main pod reconciler component
type PodReconciler struct {
	client.Client
	t                     tomb.Tomb
	Manager               ctrl.Manager
	Config                *KubernetesPod
	DisableCommandsChecks bool
	Healthcheck           *healthcheck.Component
	Logger                *zap.Logger
	Controller            controller.Controller
}

// NewPodReconciler build a pod reconciler component
func NewPodReconciler(logger *zap.Logger, healthcheck *healthcheck.Component, config *KubernetesPod, disableCommandsChecks bool) (*PodReconciler, error) {
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
		return nil, errors.Wrapf(err, "fail to create the Kubernetes pod controller manager")
	}
	reconciler := PodReconciler{
		Client:                manager.GetClient(),
		Manager:               manager,
		Logger:                logger,
		Config:                config,
		Healthcheck:           healthcheck,
		DisableCommandsChecks: disableCommandsChecks,
	}
	controller, err := controller.New("pod-controller", manager, controller.Options{
		Reconciler: &reconciler,
	})
	ctrl.SetLogger(zapr.NewLogger(logger))
	if err != nil {
		return nil, errors.Wrapf(err, "fail to create the Kubernetes pod controller")
	}
	reconciler.Controller = controller
	return &reconciler, nil
}

// Start start the pod reconciler component
func (c *PodReconciler) Start() error {

	// Watch Pods and enqueue ReplicaSet object key
	if err := c.Controller.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForObject{}); err != nil {
		c.Logger.Error(err.Error())
		return errors.Wrap(err, "fail to watch pods resources")
	}

	c.t.Go(func() error {
		ctx := c.t.Context(context.TODO())
		c.Logger.Info("starting Kubernetes pod listener")
		if err := c.Manager.Start(ctx); err != nil {
			c.Logger.Error(err.Error())
			// todo: should correctly stop the daemon if it fails
			return errors.Wrap(err, "fail to start pod manager")
		}
		c.Logger.Info("Stopping Kubernetes pod listener")
		return nil
	})
	return nil
}

// Stop stop the pod reconciler
func (c *PodReconciler) Stop() error {
	c.Logger.Info("stopping Kubernetes pod listener")
	c.t.Kill(nil)
	err := c.t.Wait()
	if err != nil {
		return err
	}
	return nil
}

// Reconcile pods healthchecks
func (c *PodReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	pods := &corev1.PodList{}
	err := c.List(ctx, pods, client.InNamespace(c.Config.Namespace), client.MatchingLabels(c.Config.Labels))
	if err != nil {
		return ctrl.Result{}, err
	}

	oldChecks := c.Healthcheck.SourceChecksNames(healthcheck.SourceKubernetesPod)
	newChecks := make(map[string]bool)
	c.Logger.Debug("Reconciling pods")

	for _, item := range pods.Items {
		phase := item.Status.Phase
		terminating := item.ObjectMeta.DeletionTimestamp != nil
		podName := item.ObjectMeta.Name
		healthcheckType := item.ObjectMeta.Annotations[typeAnnotation]
		healthcheckLabels := item.ObjectMeta.Labels
		c.Logger.Debug(fmt.Sprintf("Pod %s detected in phase %s and terminating %t", podName, phase, terminating))
		if phase == corev1.PodRunning && !terminating && healthcheckType != "" {
			healthcheckConfig := item.ObjectMeta.Annotations[configAnnotation]
			err = addCheck(c.Healthcheck, c.Logger, newChecks, healthcheckType, healthcheckConfig, item.Status.PodIP, healthcheck.SourceKubernetesPod, healthcheckLabels, c.DisableCommandsChecks)
			if err != nil {
				return ctrl.Result{}, errors.Wrapf(err, "Fail to add healthcheck for pod %s", podName)
			}
		}
	}
	err = c.Healthcheck.RemoveNonConfiguredHealthchecks(oldChecks, newChecks)
	if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
