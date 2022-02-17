package falcon

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	clog "sigs.k8s.io/controller-runtime/pkg/log"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/falcon_container_deployer"
)

// FalconContainerReconciler reconciles a FalconContainer object
type FalconContainerReconciler struct {
	client.Client
	Log        logr.Logger
	Scheme     *runtime.Scheme
	RestConfig *rest.Config
}

//+kubebuilder:rbac:groups=falcon.crowdstrike.com,resources=falconcontainers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=falcon.crowdstrike.com,resources=falconcontainers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=falcon.crowdstrike.com,resources=falconcontainers/finalizers,verbs=update

// +kubebuilder:rbac:groups=image.openshift.io,resources=imagestreams,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups="batch",resources=jobs,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;delete
// +kubebuilder:rbac:groups="",resources=pods/log,verbs=get

// +kubebuilder:rbac:groups="",resources=namespaces,verbs=create;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=create;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=create;delete
// +kubebuilder:rbac:groups="",resources=serviceaccounts,verbs=get;list;watch;patch
// +kubebuilder:rbac:groups="apps",resources=deployments,verbs=create;delete
// +kubebuilder:rbac:groups="admissionregistration.k8s.io",resources=mutatingwebhookconfigurations,verbs=create;delete
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=clusterroles,verbs=create;delete
// +kubebuilder:rbac:groups="rbac.authorization.k8s.io",resources=clusterrolebindings,verbs=create;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the FalconContainer object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.2/pkg/reconcile
func (r *FalconContainerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := clog.FromContext(ctx)
	falconContainer := &falconv1alpha1.FalconContainer{}
	err := r.Client.Get(ctx, req.NamespacedName, falconContainer)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Cannot get the Falcon Container custom resource")
		return ctrl.Result{}, err
	}

	d := falcon_container_deployer.FalconContainerDeployer{
		Ctx:        ctx,
		Client:     r.Client,
		Log:        log,
		Instance:   falconContainer.DeepCopy(),
		RestConfig: r.RestConfig,
		Scheme:     r.Scheme,
	}
	return d.Reconcile()
}

// SetupWithManager sets up the controller with the Manager.
func (r *FalconContainerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&falconv1alpha1.FalconContainer{}).
		Complete(r)
}
