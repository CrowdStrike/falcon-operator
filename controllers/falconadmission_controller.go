package controllers

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
)

// FalconAdmissionReconciler reconciles a FalconAdmission object
type FalconAdmissionReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=crowdstrike.com,resources=falconadmissions,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=crowdstrike.com,resources=falconadmissions/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=crowdstrike.com,resources=falconadmissions/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the FalconAdmission object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *FalconAdmissionReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *FalconAdmissionReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&falconv1alpha1.FalconAdmission{}).
		Complete(r)
}
