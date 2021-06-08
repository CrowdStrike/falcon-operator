package falcon

import (
	"context"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/falcon_container"
	"github.com/crowdstrike/falcon-operator/pkg/falcon_container_deployer"
)

func (r *FalconConfigReconciler) phaseBuildingReconcile(ctx context.Context, instance *falconv1alpha1.FalconConfig, logger logr.Logger) (ctrl.Result, error) {
	logger.Info("Phase: Building")
	d := falcon_container_deployer.FalconContainerDeployer{
		Ctx:      ctx,
		Client:   r.Client,
		Log:      logger,
		Instance: instance,
	}

	err := d.EnsureDockercfg(ctx, d.Namespace())
	if err != nil {
		return d.Error("Cannot find dockercfg secret from the current namespace", err)
	}
	imageStream, err := d.GetImageStream()
	if err != nil {
		return d.Error("Cannot access image stream", err)
	}

	image := falcon_container.NewImageRefresher(ctx, logger, instance.Spec.FalconAPI.ApiConfig())
	err = image.Refresh(imageStream.Status.DockerImageRepository)
	if err != nil {
		return d.Error("Cannot refresh Falcon Container Image", err)
	}
	logger.Info("Falcon Container Image pushed successfully")

	instance.Status.ErrorMessage = ""
	instance.Status.Phase = falconv1alpha1.PhaseConfiguring

	err = r.Client.Status().Update(ctx, instance)
	return ctrl.Result{}, err
}
