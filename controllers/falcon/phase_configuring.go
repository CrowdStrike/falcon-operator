package falcon

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	ctrl "sigs.k8s.io/controller-runtime"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/falcon_container_deployer"
	"github.com/crowdstrike/falcon-operator/pkg/k8s_utils"
)

const (
	JOB_NAME = "falcon-configure"
)

func (r *FalconConfigReconciler) phaseConfiguringReconcile(ctx context.Context, instance *falconv1alpha1.FalconConfig, logger logr.Logger) (ctrl.Result, error) {
	logger.Info("Phase: Configuring")
	d := falcon_container_deployer.FalconContainerDeployer{
		Ctx:      ctx,
		Client:   r.Client,
		Log:      logger,
		Instance: instance,
	}

	// (Step 1&2) Upsert Job
	job, err := d.UpsertJob()
	if err != nil {
		return d.Error("failed to upsert Job", err)
	}
	if job == nil {
		return ctrl.Result{RequeueAfter: time.Second * 5}, nil
	}

	// (Step 3) verify configuration || or re-configure job
	// TODO

	// (Step 4) wait for job completion
	if !k8s_utils.IsJobCompleted(job) {
		logger.Info("Waiting for Job completion")
		return ctrl.Result{RequeueAfter: time.Second * 5}, nil
	}

	pod, err := d.ConfigurePod()
	if err != nil {
		return d.Error("Failed to get pod relevant to configure job", err)
	}

	// (Step 5) wait for pod completion
	// TODO

	// (Step 6) obtain job output
	_, err = k8s_utils.GetPodLog(ctx, r.RestConfig, pod)
	if err != nil {
		return d.Error("Failed to get pod relevant to configure job", err)
	}

	instance.Status.ErrorMessage = ""
	instance.Status.Phase = falconv1alpha1.PhaseDeploying

	err = r.Client.Status().Update(ctx, instance)
	return ctrl.Result{}, err
}
