package falcon

import (
	"context"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
)

func (r *FalconConfigReconciler) phaseConfiguringReconcile(ctx context.Context, instance *falconv1alpha1.FalconConfig, logger logr.Logger) (ctrl.Result, error) {
	logger.Info("Phase: Configuring")

	logger.Error(nil, "TODO")

	instance.Status.ErrorMessage = ""
	instance.Status.Phase = falconv1alpha1.PhaseDone

	err := r.Client.Status().Update(ctx, instance)
	return ctrl.Result{}, err
}
