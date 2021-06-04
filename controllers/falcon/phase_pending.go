package falcon

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/falcon_container_deployer"
)

func (r *FalconConfigReconciler) phasePendingReconcile(ctx context.Context, instance *falconv1alpha1.FalconConfig, logger logr.Logger) (ctrl.Result, error) {
	logger.Info("Phase: Pending")
	d := falcon_container_deployer.FalconContainerDeployer{
		Ctx:      ctx,
		Client:   r.Client,
		Log:      logger,
		Instance: instance,
	}

	stream, err := d.UpsertImageStream()
	if err != nil {
		return d.Error("failed to upsert Image Stream", err)
	}
	if stream == nil {
		// It takes few moment for the ImageStream to be ready (shortly after it has been created)
		return ctrl.Result{RequeueAfter: time.Second * 5}, nil
	}

	instance.Status.ErrorMessage = ""
	instance.Status.Phase = falconv1alpha1.PhaseBuilding

	err = r.Client.Status().Update(ctx, instance)
	return ctrl.Result{}, err
}
