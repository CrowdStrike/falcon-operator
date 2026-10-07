package controllers

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	commonctrl "github.com/crowdstrike/falcon-operator/internal/controller/common"
	"github.com/crowdstrike/falcon-operator/internal/controller/common/image"
	"github.com/crowdstrike/falcon-operator/internal/controller/components"
	"github.com/crowdstrike/falcon-operator/internal/controller/components/clusterguard_controller"
	"github.com/crowdstrike/falcon-operator/internal/controller/components/image_analyzer"
	"github.com/crowdstrike/falcon-operator/internal/controller/components/node_sensor"
	"github.com/crowdstrike/falcon-operator/internal/controller/predicates"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	falcon_api "github.com/crowdstrike/falcon-operator/pkg/falcon_api"
	"github.com/crowdstrike/falcon-operator/version"
	"github.com/crowdstrike/gofalcon/falcon"
	"github.com/go-logr/logr"
	arv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// FalconClusterGuardReconciler reconciles a FalconClusterGuard object
type FalconClusterGuardReconciler struct {
	client.Client
	Reader        client.Reader
	RuntimeScheme *runtime.Scheme
	OpenShift     bool
	log           logr.Logger
	apiConfig     *falcon.ApiConfig
	cid           string
	cloud         string
}

// SetupWithManager sets up the controller with the Manager.
func (r *FalconClusterGuardReconciler) SetupWithManager(mgr ctrl.Manager) error {
	labelPredicate := predicates.CrowdStrikeLabel()
	generationOrLabel := predicate.Or(predicate.GenerationChangedPredicate{}, labelPredicate)
	return ctrl.NewControllerManagedBy(mgr).
		For(&falconv1alpha1.FalconClusterGuard{}, builder.WithPredicates(labelPredicate)).
		Owns(&corev1.Namespace{}, builder.WithPredicates(labelPredicate)).
		Owns(&corev1.ConfigMap{}, builder.WithPredicates(labelPredicate)).
		Owns(&corev1.Secret{}, builder.WithPredicates(labelPredicate)).
		Owns(&corev1.ServiceAccount{}, builder.WithPredicates(labelPredicate)).
		Owns(&corev1.Service{}, builder.WithPredicates(labelPredicate)).
		Owns(&rbacv1.ClusterRoleBinding{}, builder.WithPredicates(labelPredicate)).
		Owns(&rbacv1.RoleBinding{}, builder.WithPredicates(labelPredicate)).
		Owns(&appsv1.Deployment{}, builder.WithPredicates(generationOrLabel)).
		Owns(&appsv1.DaemonSet{}, builder.WithPredicates(generationOrLabel)).
		Owns(&arv1.ValidatingWebhookConfiguration{}, builder.WithPredicates(labelPredicate)).
		Complete(r)
}

// GetK8sClient returns the Kubernetes client
func (r *FalconClusterGuardReconciler) GetK8sClient() client.Client {
	return r.Client
}

// GetK8sReader returns the Kubernetes API reader
func (r *FalconClusterGuardReconciler) GetK8sReader() client.Reader {
	return r.Reader
}

// GetScheme returns the runtime scheme
func (r *FalconClusterGuardReconciler) GetScheme() *runtime.Scheme {
	return r.RuntimeScheme
}

// GetLog returns the reconciler logger
func (r *FalconClusterGuardReconciler) GetLog() logr.Logger {
	return r.log
}

//+kubebuilder:rbac:groups=falcon.crowdstrike.com,resources=falconclusterguards,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=falcon.crowdstrike.com,resources=falconclusterguards/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=falcon.crowdstrike.com,resources=falconclusterguards/finalizers,verbs=update
//+kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;delete
//+kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;delete
//+kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;delete
//+kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;delete
//+kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;delete
//+kubebuilder:rbac:groups="apps",resources=deployments,verbs=get;list;watch;create;update;delete
//+kubebuilder:rbac:groups="apps",resources=daemonsets,verbs=get;list;watch;create;update;delete
//+kubebuilder:rbac:groups="admissionregistration.k8s.io",resources=validatingwebhookconfigurations,verbs=get;list;watch;create;update;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterrolebindings,verbs=create;get;list;update;watch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=clusterroles,verbs=get;list;watch
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=rolebindings,verbs=create;get;list;update;watch;delete
//+kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles,verbs=create;get;list;update;watch;delete
//+kubebuilder:rbac:groups="coordination.k8s.io",resources=leases,verbs=get;list;watch;create;update;delete
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
func (r *FalconClusterGuardReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.log = log.FromContext(ctx)

	falconClusterGuard := &falconv1alpha1.FalconClusterGuard{}
	err := common.GetNamespacedObject(ctx, r.Client, r.Reader, req.NamespacedName, falconClusterGuard)
	if err != nil {
		if apierrors.IsNotFound(err) {
			r.log.V(1).Info("FalconClusterGuard resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}

		r.log.Error(err, "Failed to get FalconClusterGuard resource")
		return ctrl.Result{}, err
	}

	r.log.Info("Reconciling FalconClusterGuard")

	if meta.FindStatusCondition(falconClusterGuard.Status.Conditions, falconv1alpha1.ConditionSuccess) == nil {
		if err := commonctrl.StatusUpdate(ctx, r.Client, r.Status(), req, r.log, falconClusterGuard,
			falconv1alpha1.ConditionPending, metav1.ConditionFalse,
			falconv1alpha1.ReasonReqNotMet, "FalconClusterGuard progressing"); err != nil {
			return ctrl.Result{}, err
		}
	}

	if falconClusterGuard.Status.Version != version.Get() {
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			if err := common.GetNamespacedObject(ctx, r.Client, r.Reader, req.NamespacedName, falconClusterGuard); err != nil {
				return err
			}
			falconClusterGuard.Status.Version = version.Get()
			return r.Status().Update(ctx, falconClusterGuard)
		})
		if err != nil {
			r.log.Error(err, "Failed to update FalconClusterGuard status version")
			return ctrl.Result{}, err
		}
	}

	// Inject sensitive values from a k8s Secret before any reconciliation that uses them
	if falconClusterGuard.Spec.FalconSecret.Enabled {
		if err := r.injectFalconSecretData(ctx, falconClusterGuard); err != nil {
			r.log.Error(err, "Failed to inject FalconSecret data")
			return ctrl.Result{}, err
		}
	}

	if r.apiConfig == nil && falconClusterGuard.Spec.FalconAPI != nil {
		apiConfig, err := falconClusterGuard.Spec.FalconAPI.ApiConfigWithSecret(ctx, r.Reader, falconClusterGuard.Spec.FalconSecret)
		if err != nil {
			r.log.Error(err, "Failed to build Falcon API config")
			return ctrl.Result{}, err
		}
		r.apiConfig = apiConfig
	}

	if r.cid == "" {
		// CID priority: Falcon.CID (sensor spec) → FalconAPI.CID (API spec) → fetch via API credentials
		var cidOverride *string
		if falconClusterGuard.Spec.Falcon.CID != nil {
			cidOverride = falconClusterGuard.Spec.Falcon.CID
		} else if falconClusterGuard.Spec.FalconAPI != nil {
			cidOverride = falconClusterGuard.Spec.FalconAPI.CID
		}
		if cidOverride == nil && r.apiConfig == nil {
			return ctrl.Result{}, fmt.Errorf("cannot resolve CID: neither Falcon.CID nor FalconAPI credentials are configured")
		}
		cid, err := falcon_api.FalconCID(ctx, cidOverride, r.apiConfig)
		if err != nil {
			r.log.Error(err, "Failed to resolve Falcon CID")
			return ctrl.Result{}, err
		}
		r.cid = cid
	}

	imgCfg := image.Config{
		Image:              falconClusterGuard.Spec.Image,
		FalconAPI:          falconClusterGuard.Spec.FalconAPI,
		FalconSecret:       falconClusterGuard.Spec.FalconSecret,
		Version:            falconClusterGuard.Spec.Version,
		RelatedImageEnvVar: "RELATED_IMAGE_CLUSTER_GUARD",
	}

	// Image being set will override other image-based settings.
	// When a related image is set (e.g. by OLM in disconnected OpenShift environments), it takes
	// precedence over registry discovery. To pin a specific version, set Spec.Image explicitly.
	var imageURI string
	if falconClusterGuard.Spec.Image != "" || os.Getenv(imgCfg.RelatedImageEnvVar) != "" {
		var err error
		imageURI, err = image.URI(ctx, r, imgCfg, &falconClusterGuard.Status, falconClusterGuard)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to set Falcon Cloud Guard Image version: %v", err)
		}
	} else {
		if !meta.IsStatusConditionPresentAndEqual(falconClusterGuard.Status.Conditions, falconv1alpha1.ConditionImageReady, metav1.ConditionTrue) {
			uri, err := image.URI(ctx, r, imgCfg, &falconClusterGuard.Status, falconClusterGuard)
			if err != nil {
				time.Sleep(5 * time.Second)
				return ctrl.Result{RequeueAfter: 5 * time.Second}, fmt.Errorf("Cannot find Falcon Registry URI: %s", err)
			}
			r.log.Info("Skipping push of Falcon Cloud Guard image to local registry. Remote CrowdStrike registry will be used.")
			meta.SetStatusCondition(&falconClusterGuard.Status.Conditions, metav1.Condition{
				Status:             metav1.ConditionTrue,
				Reason:             falconv1alpha1.ReasonDiscovered,
				Message:            uri,
				Type:               falconv1alpha1.ConditionImageReady,
				ObservedGeneration: falconClusterGuard.GetGeneration(),
			})
			if err := r.Status().Update(ctx, falconClusterGuard); err != nil {
				return ctrl.Result{}, err
			}
			return ctrl.Result{}, nil
		}
		var err error
		imageURI, err = image.URI(ctx, r, imgCfg, &falconClusterGuard.Status, falconClusterGuard)
		if err != nil {
			time.Sleep(5 * time.Second)
			return ctrl.Result{RequeueAfter: 5 * time.Second}, err
		}
	}

	if err := commonctrl.ReconcileNamespace(ctx, r, req, falconClusterGuard, &falconClusterGuard.Status, falconClusterGuard.Spec.InstallNamespace); err != nil {
		if strings.Contains(err.Error(), "is terminating") {
			return ctrl.Result{RequeueAfter: 5 * time.Second}, nil
		}
		return ctrl.Result{}, err
	}

	imagePullSecrets := falconClusterGuard.Spec.ImagePullSecrets
	if falconClusterGuard.Spec.Registry.Type == falconv1alpha1.RegistryTypeFalconClusterGuardCrowdStrike {
		if err := r.reconcileImagePullSecret(ctx, req, falconClusterGuard); err != nil {
			return ctrl.Result{}, err
		}
		csSecret := corev1.LocalObjectReference{Name: common.FalconPullSecretName}
		if len(imagePullSecrets) == 0 || imagePullSecrets[0].Name != csSecret.Name {
			imagePullSecrets = append([]corev1.LocalObjectReference{csSecret}, imagePullSecrets...)
		}
	}

	base := components.BaseConfig{
		Request:          req,
		Owner:            falconClusterGuard,
		Status:           &falconClusterGuard.Status,
		InstallNamespace: falconClusterGuard.Spec.InstallNamespace,
		Image:            imageURI,
		ImagePullPolicy:  falconClusterGuard.Spec.ImagePullPolicy,
		ImagePullSecrets: imagePullSecrets,
		Cid:              r.cid,
		Falcon:           falconClusterGuard.Spec.Falcon,
		OpenShift:        r.OpenShift,
	}

	// Build every component config before any of them runs: status updates re-fetch the CR into
	// falconClusterGuard, which drops data injected from the FalconSecret.
	admission := clusterguard_controller.New(r, clusterguard_controller.Config{
		BaseConfig:                   base,
		ClusterGuardControllerConfig: falconClusterGuard.Spec.ClusterGuardControllerConfig,
		ClusterName:                  falconClusterGuard.GetClusterName(),
		RegistryTLS:                  falconClusterGuard.Spec.Registry.TLS,
		ImageAnalyzerEnabled:         falconClusterGuard.Spec.ImageAnalyzer.IsEnabled(),
	})
	nodeSensor := node_sensor.New(r, node_sensor.Config{
		BaseConfig: base,
		FalconAPI:  falconClusterGuard.Spec.FalconAPI,
		NodeSensor: falconClusterGuard.Spec.NodeSensor,
	})
	imageAnalyzer := image_analyzer.New(r, image_analyzer.Config{
		BaseConfig:        base,
		ImageAnalyzerSpec: falconClusterGuard.Spec.ImageAnalyzer,
		FalconAPI:         falconClusterGuard.Spec.FalconAPI,
	})
	admissionEnabled := falconClusterGuard.Spec.ClusterGuardControllerConfig.IsEnabled()
	nodeSensorEnabled := falconClusterGuard.Spec.NodeSensor.IsEnabled()
	imageAnalyzerEnabled := falconClusterGuard.Spec.ImageAnalyzer.IsEnabled()

	if falconClusterGuard.GetDeletionTimestamp() != nil {
		if !controllerutil.ContainsFinalizer(falconClusterGuard, common.FalconFinalizer) {
			return ctrl.Result{}, nil
		}
		r.log.Info("FalconClusterGuard is being deleted, running finalization logic")
		finalize := nodeSensor.Cleanup
		if nodeSensorEnabled {
			finalize = nodeSensor.Finalize
		}
		if result, err := finalize(ctx); err != nil || result.RequeueAfter > 0 {
			return result, err
		}
		// Removing the finalizer lets the API server delete the CR, so no status update follows.
		return ctrl.Result{}, r.setFinalizer(ctx, falconClusterGuard, false)
	}

	// The finalizer guarantees node cleanup runs before the CR, and with it the sensor DaemonSet, is deleted.
	if nodeSensorEnabled {
		if err := r.setFinalizer(ctx, falconClusterGuard, true); err != nil {
			return ctrl.Result{}, err
		}
	}

	if err := r.reconcileAPITLSSecrets(ctx, req, falconClusterGuard); err != nil {
		return ctrl.Result{}, err
	}

	// Disabled components are cleaned up in place; Cleanup is a no-op once their resources are gone.
	if admissionEnabled {
		if result, err := admission.Reconcile(ctx); err != nil || result.RequeueAfter > 0 {
			return result, err
		}
	} else if err := admission.Cleanup(ctx); err != nil {
		return ctrl.Result{}, err
	}

	if nodeSensorEnabled {
		if result, err := nodeSensor.Reconcile(ctx); err != nil || result.RequeueAfter > 0 {
			return result, err
		}
	} else {
		if result, err := nodeSensor.Cleanup(ctx); err != nil || result.RequeueAfter > 0 {
			return result, err
		}
		if err := r.setFinalizer(ctx, falconClusterGuard, false); err != nil {
			return ctrl.Result{}, err
		}
	}

	if imageAnalyzerEnabled {
		if result, err := imageAnalyzer.Reconcile(ctx); err != nil || result.RequeueAfter > 0 {
			return result, err
		}
	} else if err := imageAnalyzer.Cleanup(ctx); err != nil {
		return ctrl.Result{}, err
	}

	if err := commonctrl.StatusUpdate(ctx, r.Client, r.Status(), req, r.log, falconClusterGuard,
		falconv1alpha1.ConditionSuccess, metav1.ConditionTrue,
		falconv1alpha1.ReasonInstallSucceeded, "FalconClusterGuard installation completed"); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
