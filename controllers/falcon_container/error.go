package falcon

import (
	"context"

	"github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *FalconContainerReconciler) Error(ctx context.Context, req ctrl.Request, falconContainer *v1alpha1.FalconContainer, message string) {
	r.updateStatus(ctx, req, falconContainer, v1alpha1.PhaseError, message)
}

func (r *FalconContainerReconciler) UpdateStatus(ctx context.Context, req ctrl.Request, falconContainer *v1alpha1.FalconContainer, phase v1alpha1.FalconContainerStatusPhase) {
	r.updateStatus(ctx, req, falconContainer, phase, "")
}

func (r *FalconContainerReconciler) updateStatus(ctx context.Context, req ctrl.Request, falconContainer *v1alpha1.FalconContainer, phase v1alpha1.FalconContainerStatusPhase, message string) {
	if err := r.updateCRStatus(ctx, req, falconContainer); err != nil {
		r.Log.Error(err, "cannot refresh the Falcon Container custom resource")
		return
	}
	update := false
	if falconContainer.Status.Phase != phase {
		falconContainer.Status.Phase = phase
		update = true
	}
	if falconContainer.Status.ErrorMessage != message {
		falconContainer.Status.ErrorMessage = message
		update = true
	}
	if update {
		r.Log.Info("Updating Falcon Container CR status")
		if err := r.Client.Status().Update(ctx, falconContainer); err != nil {
			r.Log.Error(err, "failed to update Falcon Container Status")
		}
	}
}
func (r *FalconContainerReconciler) updateCRStatus(ctx context.Context, req ctrl.Request, falconContainer *v1alpha1.FalconContainer) error {
	return r.Client.Get(ctx, req.NamespacedName, falconContainer)
}
