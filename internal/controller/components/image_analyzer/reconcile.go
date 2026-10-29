package image_analyzer

import (
	"context"
	"maps"
	"strconv"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	k8sutils "github.com/crowdstrike/falcon-operator/internal/controller/common"
	"github.com/crowdstrike/falcon-operator/internal/controller/components"
	pkgcommon "github.com/crowdstrike/falcon-operator/pkg/common"
	appsv1 "k8s.io/api/apps/v1"
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

// withAppLabels adds the Helm chart's image analyzer app.kubernetes.io labels to l.
func (ia *ImageAnalyzer) withAppLabels(l map[string]string) map[string]string {
	maps.Copy(l, pkgcommon.AppLabels(pkgcommon.ClusterGuardImageAnalyzerAppName, ia.cfg.Request.Name, pkgcommon.FalconImageAnalyzerComponentName))
	return l
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

	// A Deployment that is not ready yet is reported through the condition rather than a requeue, so the
	// remaining components still reconcile; the FalconClusterGuard controller requeues until it is ready.
	status, err := k8sutils.DeploymentStatus(ctx, ia.r,
		types.NamespacedName{Name: pkgcommon.FCGImageAnalyzerDeploymentName, Namespace: ia.cfg.InstallNamespace})
	if err != nil {
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, k8sutils.ConditionsSet(ia.r, ctx, ia.cfg.Request, ia.r.GetLog(), ia.cfg.Owner, ia.cfg.Status,
		status.Condition(falconv1alpha1.ConditionImageAnalyzerReady, "Image Analyzer is ready", ia.cfg.Owner.GetGeneration()))
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
