/*
Copyright 2021 CrowdStrike
*/

package falcon

import (
	"context"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
)

// FalconConfigReconciler reconciles a FalconConfig object
type FalconConfigReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=falcon.crowdstrike.com,resources=falconconfigs,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=falcon.crowdstrike.com,resources=falconconfigs/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=falcon.crowdstrike.com,resources=falconconfigs/finalizers,verbs=update
// +kubebuilder:rbac:groups=image.openshift.io,resources=imagestreams,verbs=get;list;watch;create;update;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the FalconConfig object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.7.0/pkg/reconcile
func (r *FalconConfigReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := r.Log.WithValues("falconconfig", req.NamespacedName)
	logger.Info("Reconciling FalconConfig")

	// your logic here
	falconConfig := &falconv1alpha1.FalconConfig{}
	err := r.Client.Get(ctx, req.NamespacedName, falconConfig)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		logger.Error(err, "Cannot get the Falcon Config")
		return ctrl.Result{}, err
	}

	instanceToBeUpdated := falconConfig.DeepCopy()

	if instanceToBeUpdated.Status.Phase == "" {
		instanceToBeUpdated.Status.Phase = falconv1alpha1.PhasePending
	}

	switch instanceToBeUpdated.Status.Phase {
	case falconv1alpha1.PhasePending:
		return r.phasePendingReconcile(ctx, instanceToBeUpdated, logger)
	case falconv1alpha1.PhaseBuilding:
		return r.phaseBuildingReconcile(ctx, instanceToBeUpdated, logger)
	case falconv1alpha1.PhaseConfiguring:
		return r.phaseConfiguringReconcile(ctx, instanceToBeUpdated, logger)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *FalconConfigReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&falconv1alpha1.FalconConfig{}).
		Complete(r)
}
