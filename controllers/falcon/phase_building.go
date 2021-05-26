package falcon

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/falcon_container"
)

func (r *FalconConfigReconciler) phaseBuildingReconcile(ctx context.Context, instance *falconv1alpha1.FalconConfig, logger logr.Logger) (ctrl.Result, error) {
	logger.Info("Phase: Building")

	refreshImage, err := r.reconcileContainerImage(instance)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("Error when reconciling Falcon Container Image: %w", err)
	}
	if refreshImage {
		err = r.refreshContainerImage(ctx, instance)
		if err != nil {
			return ctrl.Result{}, fmt.Errorf("Error when reconciling Falcon Container Image: %w", err)
		}
		// TODO: write status
	}

	instance.Status.Phase = falconv1alpha1.PhaseDone

	err = r.Client.Status().Update(ctx, instance)
	return ctrl.Result{}, err
}

func (r *FalconConfigReconciler) refreshContainerImage(ctx context.Context, falconConfig *falconv1alpha1.FalconConfig) error {
	image := falcon_container.NewImageRefresher(ctx, r.Log, falconConfig.Spec.FalconAPI.ApiConfig())
	return image.Refresh(falconConfig.Spec.WorkloadProtectionSpec.LinuxContainerSpec.Registry)
}

func (r *FalconConfigReconciler) reconcileContainerImage(falconConfig *falconv1alpha1.FalconConfig) (bool, error) {
	if falconConfig.Status.WorkloadProtectionStatus == nil {
		return true, nil
	}
	return false, nil
}
