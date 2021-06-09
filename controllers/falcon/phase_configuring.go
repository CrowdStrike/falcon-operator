package falcon

import (
	"context"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/falcon_container_deployer"
)

const (
	JOB_NAME = "falcon-configure"
)

func (r *FalconConfigReconciler) phaseConfiguringReconcile(ctx context.Context, instance *falconv1alpha1.FalconConfig, logger logr.Logger) (ctrl.Result, error) {
	logger.Info("Phase: Configuring")
	d := falcon_container_deployer.FalconContainerDeployer{
		Ctx:        ctx,
		Client:     r.Client,
		Log:        logger,
		Instance:   instance,
		RestConfig: r.RestConfig,
	}

	return d.PhaseConfiguring()
}
