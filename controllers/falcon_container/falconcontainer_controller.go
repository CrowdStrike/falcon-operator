package falcon

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	arv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clog "sigs.k8s.io/controller-runtime/pkg/log"

	"github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
)

// FalconContainerReconciler reconciles a FalconContainer object
type FalconContainerReconciler struct {
	client.Client
	Log        logr.Logger
	Scheme     *runtime.Scheme
	RestConfig *rest.Config
}

//+kubebuilder:rbac:groups=falcon.crowdstrike.com,resources=falconcontainers,verbs=get;list;watch;create;update;patch;delete

// +kubebuilder:rbac:groups=image.openshift.io,resources=imagestreams,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=deployments,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="apps",resources=deployments,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="admissionregistration.k8s.io",resources=mutatingwebhookconfigurations,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=clusterroles,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=clusterrolebindings,verbs=get;list;watch;create;update;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *FalconContainerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	r.Log = clog.FromContext(ctx)
	falconContainer := &v1alpha1.FalconContainer{}

	err := r.Client.Get(ctx, req.NamespacedName, falconContainer)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the objectclusterRole := r.newClusterRole() - requeue the request.
		r.Log.Error(err, "cannot get the Falcon Container custom resource")
		return ctrl.Result{}, err
	}

	r.Log.Info("Reconciling Namespace")
	if _, err = r.reconcileNamespace(ctx, falconContainer); err != nil {
		return ctrl.Result{}, err
	}

	switch falconContainer.Spec.Registry.Type {
	case v1alpha1.RegistryTypeECR:
		r.Log.Info("Reconciling ECR Repository")
		if _, err := r.UpsertECRRepo(ctx); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to create ECR repository: %v", err)
		}
	case v1alpha1.RegistryTypeOpenshift:
		r.Log.Info("Reconciling Image Stream")
		stream, err := r.reconcileImageStream(ctx, falconContainer)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to create Image Stream")
		}
		if stream == nil {
			return ctrl.Result{}, nil
		}
	}
	if r.imageMirroringEnabled(falconContainer) {
		r.Log.Info("Verifying image availability in remote registry")
		if err = r.PushImage(ctx, falconContainer); err != nil {
			return ctrl.Result{}, fmt.Errorf("cannot refresh Falcon Container image: %v", err)
		}
	} else {
		r.Log.Info("Verifying access to CrowdStrike Container Image Registry")
		updated, err := r.verifyCrowdStrikeRegistry(ctx, falconContainer)
		if updated {
			return ctrl.Result{}, nil
		}
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to verify CrowdStrike Container Image Registry access")
		}
		r.Log.Info("Reconciling Container Registry pull token Secrets")
		if _, err = r.reconcileRegistrySecrets(ctx, falconContainer); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to create Falcon registry pull token Secrets: %v", err)
		}
	}

	r.Log.Info("Reconciling ServiceAccount")
	if _, err = r.reconcileServiceAccount(ctx, falconContainer); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to reconcile Service Account: %v", err)
	}

	r.Log.Info("Reconciling Cluster Role")
	if _, err = r.reconcileClusterRole(ctx, falconContainer); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to reconcile Cluster Role: %v", err)
	}

	r.Log.Info("Reconciling Cluster Role Binding")
	if _, err = r.reconcileClusterRoleBinding(ctx, falconContainer); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to reconcile Cluster Role Binding: %v", err)
	}

	r.Log.Info("Reconciling injector webhook TLS Secret")
	injectorTLS, err := r.reconcileInjectorTLSSecret(ctx, falconContainer)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to reconcile injector TLS Secret: %v", err)
	}
	caBundle := injectorTLS.Data["ca.crt"]
	if caBundle == nil {
		return ctrl.Result{}, fmt.Errorf("CA bundle not present in injector TLS Secret")
	}

	r.Log.Info("Reconciling injector ConfigMap")
	if _, err = r.reconcileConfigMap(ctx, falconContainer); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to reconcile injector ConfigMap: %v", err)
	}

	r.Log.Info("Reconciling injector Deployment")
	if _, err = r.reconcileDeployment(ctx, falconContainer); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to reconcile injector Deployment: %v", err)
	}

	r.Log.Info("Reconciling injector Service")
	if _, err = r.reconcileService(ctx, falconContainer); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to reconcile injector Service: %v", err)
	}

	r.Log.Info("Ensuring injector pod is in Ready state")
	if _, err = r.injectorPodReady(ctx, falconContainer); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to find Ready injector pod: %v", err)
	}

	r.Log.Info("Reconciling injector Mutating Webhook Configuration")
	if _, err = r.reconcileWebhook(ctx, falconContainer, caBundle); err != nil {
		return ctrl.Result{}, fmt.Errorf("failed to reconcile injector MutatingWebhookConfiguration: %v", err)
	}

	return ctrl.Result{}, nil

}

// SetupWithManager sets up the controller with the Manager.
func (r *FalconContainerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.FalconContainer{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&corev1.Service{}).
		Owns(&rbacv1.ClusterRole{}).
		Owns(&rbacv1.ClusterRoleBinding{}).
		Owns(&arv1.MutatingWebhookConfiguration{}).
		Complete(r)
}
