package falcon

import (
	"context"

	"github.com/go-logr/logr"
	batchv1 "k8s.io/api/batch/v1"
	types "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/falcon_container_deployer"
	"github.com/crowdstrike/falcon-operator/pkg/k8s_utils"
)

func (r *FalconConfigReconciler) phaseDeployingReconcile(ctx context.Context, instance *falconv1alpha1.FalconConfig, logger logr.Logger) (ctrl.Result, error) {
	d := falcon_container_deployer.FalconContainerDeployer{
		Ctx:      ctx,
		Client:   r.Client,
		Log:      logger,
		Instance: instance,
	}

	logger.Info("Phase: Deploying")

	namespace := instance.ObjectMeta.Namespace

	job := &batchv1.Job{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: JOB_NAME, Namespace: namespace}, job)
	if err != nil {
		return d.Error("Failed to get Job", err)
	}

	pod, err := r.configurePod(ctx, instance, job, logger)
	if err != nil {
		return d.Error("Failed to get pod relevant to configure job", err)
	}

	yaml, err := k8s_utils.GetPodLog(ctx, r.RestConfig, pod)
	if err != nil {
		return d.Error("Failed to get pod relevant to configure job", err)
	}

	objects, err := k8s_utils.ParseK8sObjects(yaml)
	if err != nil {
		return d.Error("Failed to parse output of installer", err)
	}

	err = k8s_utils.Create(ctx, r.Client, objects, logger)
	if err != nil {
		return d.Error("Failed to create Falcon Container objects in the cluster", err)
	}

	instance.Status.ErrorMessage = ""
	instance.Status.Phase = falconv1alpha1.PhaseDone

	err = r.Client.Status().Update(ctx, instance)
	return ctrl.Result{}, err
}
