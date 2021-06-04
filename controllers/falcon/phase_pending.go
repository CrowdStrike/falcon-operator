package falcon

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
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

	_, err := d.GetImageStream()
	if err != nil && errors.IsNotFound(err) {
		err := d.CreateImageStream()
		if err != nil {
			return d.Error("failed to create Image Stream", err)
		}
		// It takes few moment for the ImageStream to be ready
		return ctrl.Result{RequeueAfter: time.Second * 5}, nil

	} else if err != nil {
		return d.Error("Failed to get ImageStream", err)
	}

	instance.Status.ErrorMessage = ""
	instance.Status.Phase = falconv1alpha1.PhaseBuilding

	err = r.Client.Status().Update(ctx, instance)
	return ctrl.Result{}, err
}
