package kubernetes

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/go-logr/zapr"
	cabourottemcorbinfrv1 "github.com/mcorbin/cabourotte/api/v1"
	"github.com/mcorbin/cabourotte/healthcheck"
	"github.com/pkg/errors"
	"gopkg.in/tomb.v2"
)

// HealthcheckReconciler reconciles a Healthcheck object
type HealthcheckReconciler struct {
	client.Client
	Scheme      *runtime.Scheme
	t           tomb.Tomb
	Manager     ctrl.Manager
	Healthcheck *healthcheck.Component
	Config      *KubernetesCRD
	Logger      *zap.Logger
}

// NewHealthcheckReconciler build a pod reconciler component
func NewHealthcheckReconciler(logger *zap.Logger, healthcheck *healthcheck.Component, config *KubernetesCRD) (*HealthcheckReconciler, error) {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(cabourottemcorbinfrv1.AddToScheme(scheme))

	kubeConfig, err := ctrl.GetConfig()
	if err != nil {
		return nil, errors.Wrapf(err, "fail to get the Kubernetes client configuration")
	}
	manager, err := ctrl.NewManager(kubeConfig,
		ctrl.Options{
			Scheme:             scheme,
			Namespace:          config.Namespace,
			MetricsBindAddress: "0",
		})
	if err != nil {
		return nil, errors.Wrapf(err, "fail to create the Kubernetes pod controller manager")
	}
	reconciler := HealthcheckReconciler{
		Client:      manager.GetClient(),
		Scheme:      manager.GetScheme(),
		Manager:     manager,
		Logger:      logger,
		Config:      config,
		Healthcheck: healthcheck,
	}
	if err = reconciler.SetupWithManager(manager); err != nil {

		return nil, errors.Wrapf(err, "fail to setup the kubernetes healthcheck controller")
	}
	ctrl.SetLogger(zapr.NewLogger(logger))
	if err != nil {
		return nil, errors.Wrapf(err, "fail to create the Kubernetes pod controller")
	}
	return &reconciler, nil
}

// Start start the pod reconciler component
func (c *HealthcheckReconciler) Start() error {
	c.t.Go(func() error {
		ctx := c.t.Context(context.TODO())
		c.Logger.Info("starting Kubernetes healthcheck listener")
		if err := c.Manager.Start(ctx); err != nil {
			c.Logger.Error(err.Error())
			// todo: should correctly stop the daemon if it fails
			return errors.Wrap(err, "fail to start healthcheck manager")
		}
		c.Logger.Info("Stopping Kubernetes healthcheck listener")
		return nil
	})
	return nil
}

// Stop stop the pod reconciler
func (c *HealthcheckReconciler) Stop() error {
	c.Logger.Info("stopping Kubernetes healthcheck listener")
	c.t.Kill(nil)
	err := c.t.Wait()
	if err != nil {
		return err
	}
	return nil
}

func (c *HealthcheckReconciler) reconcileCRDs(crd *cabourottemcorbinfrv1.HealthcheckList) (ctrl.Result, error) {
	c.Logger.Debug("Reconciling CRD")
	newChecks := make(map[string]bool)
	oldChecks := c.Healthcheck.SourceChecksNames(healthcheck.SourceKubernetesCRD)
	for _, item := range crd.Items {
		crdName := item.ObjectMeta.Name
		c.Logger.Info(fmt.Sprintf("Reconciling healthcheck CRD %s", crdName))
		checksLabels := item.ObjectMeta.Labels
		for _, dnsCheckConfig := range item.Spec.DNSChecks {
			err := dnsCheckConfig.Validate()
			if err != nil {
				return ctrl.Result{}, err
			}
			mergeLabels(&dnsCheckConfig.Base, checksLabels)
			dnsCheckConfig.Base.Source = healthcheck.SourceKubernetesCRD
			check := healthcheck.NewDNSHealthcheck(c.Logger, &dnsCheckConfig)
			err = c.Healthcheck.AddCheck(check)
			if err != nil {
				return ctrl.Result{}, err
			}
			newChecks[dnsCheckConfig.Base.Name] = true
		}
		for _, commandCheckConfig := range item.Spec.CommandChecks {
			err := commandCheckConfig.Validate()
			if err != nil {
				return ctrl.Result{}, err
			}
			mergeLabels(&commandCheckConfig.Base, checksLabels)
			commandCheckConfig.Base.Source = healthcheck.SourceKubernetesCRD
			check := healthcheck.NewCommandHealthcheck(c.Logger, &commandCheckConfig)
			err = c.Healthcheck.AddCheck(check)
			if err != nil {
				return ctrl.Result{}, err
			}
			newChecks[commandCheckConfig.Base.Name] = true
		}
		for _, tcpCheckConfig := range item.Spec.TCPChecks {
			err := tcpCheckConfig.Validate()
			if err != nil {
				return ctrl.Result{}, err
			}
			mergeLabels(&tcpCheckConfig.Base, checksLabels)
			tcpCheckConfig.Base.Source = healthcheck.SourceKubernetesCRD
			check := healthcheck.NewTCPHealthcheck(c.Logger, &tcpCheckConfig)
			err = c.Healthcheck.AddCheck(check)
			if err != nil {
				return ctrl.Result{}, err
			}
			newChecks[tcpCheckConfig.Base.Name] = true
		}
		for _, httpCheckConfig := range item.Spec.HTTPChecks {
			err := httpCheckConfig.Validate()
			if err != nil {
				return ctrl.Result{}, err
			}
			mergeLabels(&httpCheckConfig.Base, checksLabels)
			httpCheckConfig.Base.Source = healthcheck.SourceKubernetesCRD
			check := healthcheck.NewHTTPHealthcheck(c.Logger, &httpCheckConfig)
			err = c.Healthcheck.AddCheck(check)
			if err != nil {
				return ctrl.Result{}, err
			}
			newChecks[httpCheckConfig.Base.Name] = true
		}
		for _, tlsCheckConfig := range item.Spec.TLSChecks {
			err := tlsCheckConfig.Validate()
			if err != nil {
				return ctrl.Result{}, err
			}
			mergeLabels(&tlsCheckConfig.Base, checksLabels)
			tlsCheckConfig.Base.Source = healthcheck.SourceKubernetesCRD
			check := healthcheck.NewTLSHealthcheck(c.Logger, &tlsCheckConfig)
			err = c.Healthcheck.AddCheck(check)
			if err != nil {
				return ctrl.Result{}, err
			}
			newChecks[tlsCheckConfig.Base.Name] = true
		}
	}
	err := c.Healthcheck.RemoveNonConfiguredHealthchecks(oldChecks, newChecks)
	if err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil

}

//+kubebuilder:rbac:groups=cabourotte.mcorbin.fr,resources=healthchecks,verbs=get;list;watch
//+kubebuilder:rbac:groups=cabourotte.mcorbin.fr,resources=healthchecks/status,verbs=get
func (c *HealthcheckReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	crd := &cabourottemcorbinfrv1.HealthcheckList{}
	err := c.List(ctx, crd, client.InNamespace(c.Config.Namespace), client.MatchingLabels(c.Config.Labels))
	if err != nil {
		return ctrl.Result{}, err
	}
	return c.reconcileCRDs(crd)
}

// SetupWithManager sets up the controller with the Manager.
func (c *HealthcheckReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&cabourottemcorbinfrv1.Healthcheck{}).
		Complete(c)
}
