package falcon

import (
	"context"

	"github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
)

func (r *FalconContainerReconciler) Error(ctx context.Context, falconContainer *v1alpha1.FalconContainer, message string) {
	r.updateStatus(ctx, falconContainer, v1alpha1.PhaseError, message)
}

func (r *FalconContainerReconciler) UpdateStatus(ctx context.Context, falconContainer *v1alpha1.FalconContainer, phase v1alpha1.FalconContainerStatusPhase) {
	r.updateStatus(ctx, falconContainer, phase, "")
}

func (r *FalconContainerReconciler) updateStatus(ctx context.Context, falconContainer *v1alpha1.FalconContainer, phase v1alpha1.FalconContainerStatusPhase, message string) {
	falconContainer.Status.Phase = phase
	falconContainer.Status.ErrorMessage = message
	if err := r.Client.Status().Update(ctx, falconContainer); err != nil {
		r.Log.Error(err, "failed to update Falcon Container Status")
	}
}
