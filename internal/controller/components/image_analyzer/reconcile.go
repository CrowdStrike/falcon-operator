package image_analyzer

import (
	"context"
	"strconv"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	k8sutils "github.com/crowdstrike/falcon-operator/internal/controller/common"
	"github.com/crowdstrike/falcon-operator/internal/controller/components"
	pkgcommon "github.com/crowdstrike/falcon-operator/pkg/common"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

// Config holds the inputs needed to reconcile the FCG Image Analyzer component.
type Config struct {
	components.BaseConfig
	ImageAnalyzerSpec falconv1alpha1.FalconClusterGuardImageAnalyzerSpec
	FalconAPI         *falconv1alpha1.FalconAPI
}

// ImageAnalyzer owns the reconciliation of all Image Analyzer sub-resources.
type ImageAnalyzer struct {
	r   k8sutils.Reconciler
	cfg Config
}

// New returns an ImageAnalyzer ready to reconcile.
func New(r k8sutils.Reconciler, cfg Config) *ImageAnalyzer {
	return &ImageAnalyzer{r: r, cfg: cfg}
}

// Reconcile runs all Image Analyzer reconciliation steps in order.
func (ia *ImageAnalyzer) Reconcile(ctx context.Context) (ctrl.Result, error) {
	if err := ia.reconcileServiceAccount(ctx); err != nil {
		return ctrl.Result{}, err
	}
	if err := ia.reconcileClusterRoleBinding(ctx); err != nil {
		return ctrl.Result{}, err
	}
	configUpdated, err := ia.reconcileConfigMap(ctx)
	if err != nil {
		return ctrl.Result{}, err
	}
	if err := ia.reconcileTLSSecret(ctx); err != nil {
		return ctrl.Result{}, err
	}
	if err := ia.reconcileService(ctx); err != nil {
		return ctrl.Result{}, err
	}
	if err := ia.reconcileDeployment(ctx); err != nil {
		return ctrl.Result{}, err
	}

	if configUpdated {
		if err := ia.triggerRollingRestart(ctx); err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, k8sutils.ConditionsUpdate(ia.r, ctx, ia.cfg.Request, ia.r.GetLog(), ia.cfg.Owner, ia.cfg.Status, metav1.Condition{
		Type:               falconv1alpha1.ConditionImageAnalyzerReady,
		Status:             metav1.ConditionTrue,
		Reason:             falconv1alpha1.ReasonInstallSucceeded,
		Message:            "Image Analyzer is ready",
		ObservedGeneration: ia.cfg.Owner.GetGeneration(),
	})
}

// triggerRollingRestart bumps the config-version annotation on the Deployment to force a rolling restart.
func (ia *ImageAnalyzer) triggerRollingRestart(ctx context.Context) error {
	log := ia.r.GetLog()
	existing := &appsv1.Deployment{}
	if err := pkgcommon.GetNamespacedObject(ctx, ia.r, ia.r.GetK8sReader(),
		types.NamespacedName{Name: pkgcommon.FCGImageAnalyzerDeploymentName, Namespace: ia.cfg.InstallNamespace}, existing); err != nil {
		log.Error(err, "Failed to get FCG Image Analyzer Deployment for rolling restart")
		return err
	}

	const configVersion = "falcon.config.version"
	if existing.Spec.Template.Annotations == nil {
		existing.Spec.Template.Annotations = make(map[string]string)
	}
	if v, ok := existing.Spec.Template.Annotations[configVersion]; ok {
		i, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		existing.Spec.Template.Annotations[configVersion] = strconv.Itoa(i + 1)
	} else {
		existing.Spec.Template.Annotations[configVersion] = "1"
	}

	log.Info("Rolling FCG Image Analyzer Deployment due to config change")
	existing.SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind("Deployment"))
	return k8sutils.Update(ia.r, ctx, ia.cfg.Request, log, ia.cfg.Owner, ia.cfg.Status, existing)
}
