package clusterguard_controller

import (
	"context"
	"time"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	k8sutils "github.com/crowdstrike/falcon-operator/internal/controller/common"
	"github.com/crowdstrike/falcon-operator/internal/controller/components"
	pkgcommon "github.com/crowdstrike/falcon-operator/pkg/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// Config holds the inputs needed to reconcile the admission controller component.
type Config struct {
	components.BaseConfig
	ClusterGuardControllerConfig falconv1alpha1.FalconClusterGuardController
	ClusterName                  *string
	RegistryTLS                  falconv1alpha1.RegistryTLSSpec
	// ImageAnalyzerEnabled indicates IAR is deployed by FalconClusterGuard in InstallNamespace,
	// which overrides ClusterGuardControllerConfig.FalconImageAnalyzerNamespace.
	ImageAnalyzerEnabled bool
}

// Admission owns the reconciliation of all admission controller sub-resources.
type ClusterGuardController struct {
	r   k8sutils.Reconciler
	cfg Config
}

// New returns an Admission ready to reconcile.
func New(r k8sutils.Reconciler, cfg Config) *ClusterGuardController {
	return &ClusterGuardController{r: r, cfg: cfg}
}

// Reconcile runs all admission controller reconciliation steps in order.
func (a *ClusterGuardController) Reconcile(ctx context.Context) (ctrl.Result, error) {
	log := a.r.GetLog()

	if err := a.reconcileServiceAccount(ctx); err != nil {
		return ctrl.Result{}, err
	}
	if err := a.reconcileClusterRoleBinding(ctx); err != nil {
		return ctrl.Result{}, err
	}
	if err := a.reconcileRole(ctx); err != nil {
		return ctrl.Result{}, err
	}
	if err := a.reconcileRoleBinding(ctx); err != nil {
		return ctrl.Result{}, err
	}
	if err := a.reconcileResourceQuota(ctx); err != nil {
		return ctrl.Result{}, err
	}
	configUpdated, err := a.reconcileConfigMap(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}
	clusterNameConfigUpdated, err := a.reconcileClusterNameConfigMap(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}
	tlsSecret, err := a.reconcileTLSSecret(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}
	webhookServiceUpdated, err := a.reconcileWebhookService(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}
	apiServiceUpdated, err := a.reconcileAPIService(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}
	webhookUpdated, err := a.reconcileValidatingWebhook(ctx, tlsSecret.Data["ca.crt"])
	if err != nil {
		return ctrl.Result{}, err
	}
	if err := a.reconcileDeployment(ctx); err != nil {
		return ctrl.Result{}, err
	}

	if configUpdated || clusterNameConfigUpdated || webhookServiceUpdated || apiServiceUpdated || webhookUpdated {
		pod, err := k8sutils.GetReadyPod(a.r.GetK8sReader(), ctx, a.cfg.InstallNamespace,
			client.MatchingLabels{"app": pkgcommon.ClusterGuardControllerServiceApp})
		if err != nil && err != k8sutils.ErrNoWebhookServicePodReady {
			log.Error(err, "Failed to find Ready FalconClusterGuard pod")
			return ctrl.Result{}, err
		}
		if pod.Name == "" {
			log.Info("Looking for a Ready FalconClusterGuard pod", "namespace", a.cfg.InstallNamespace)
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}
		return ctrl.Result{}, a.triggerRollingDeployment(ctx)
	}

	return ctrl.Result{}, k8sutils.ConditionsUpdate(a.r, ctx, a.cfg.Request, log, a.cfg.Owner, a.cfg.Status, metav1.Condition{
		Type:               falconv1alpha1.ConditionAdmissionReady,
		Status:             metav1.ConditionTrue,
		Reason:             falconv1alpha1.ReasonInstallSucceeded,
		Message:            "Admission controller is ready",
		ObservedGeneration: a.cfg.Owner.GetGeneration(),
	})
}
