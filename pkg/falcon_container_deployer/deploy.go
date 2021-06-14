package falcon_container_deployer

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	"k8s.io/client-go/rest"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/k8s_utils"
)

type FalconContainerDeployer struct {
	Ctx context.Context
	client.Client
	Log        logr.Logger
	Instance   *falconv1alpha1.FalconContainer
	RestConfig *rest.Config
}

func (d *FalconContainerDeployer) Reconcile() (ctrl.Result, error) {
	if d.Instance.Status.Phase == "" {
		d.Instance.Status.Phase = falconv1alpha1.PhasePending
	}

	d.Log.Info("Falcon Container Deploy", "Phase", d.Instance.Status.Phase)
	switch d.Instance.Status.Phase {
	case falconv1alpha1.PhasePending:
		return d.PhasePending()
	case falconv1alpha1.PhaseBuilding:
		return d.PhaseBuilding()
	case falconv1alpha1.PhaseConfiguring:
		return d.PhaseConfiguring()
	case falconv1alpha1.PhaseDeploying:
		return d.PhaseDeploying()
	}

	return ctrl.Result{}, nil

}

func (d *FalconContainerDeployer) PhasePending() (ctrl.Result, error) {
	stream, err := d.UpsertImageStream()
	if err != nil {
		return d.Error("failed to upsert Image Stream", err)
	}
	if stream == nil {
		// It takes few moment for the ImageStream to be ready (shortly after it has been created)
		return ctrl.Result{RequeueAfter: time.Second * 5}, nil
	}

	return d.NextPhase(falconv1alpha1.PhaseBuilding)
}

func (d *FalconContainerDeployer) PhaseBuilding() (ctrl.Result, error) {
	err := d.EnsureDockercfg()
	if err != nil {
		return d.Error("Cannot find dockercfg secret from the current namespace", err)
	}
	err = d.PushImage()
	if err != nil {
		return d.Error("Cannot refresh Falcon Container image", err)
	}

	return d.NextPhase(falconv1alpha1.PhaseConfiguring)
}

func (d *FalconContainerDeployer) PhaseConfiguring() (ctrl.Result, error) {
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
		d.Log.Info("Waiting for Job completion")
		return ctrl.Result{RequeueAfter: time.Second * 5}, nil
	}

	pod, err := d.ConfigurePod()
	if err != nil {
		return d.Error("Failed to get pod relevant to configure job", err)
	}

	// (Step 5) wait for pod completion
	// TODO

	// (Step 6) obtain job output
	_, err = k8s_utils.GetPodLog(d.Ctx, d.RestConfig, pod)
	if err != nil {
		return d.Error("Failed to get pod relevant to configure job", err)
	}

	return d.NextPhase(falconv1alpha1.PhaseDeploying)
}

func (d *FalconContainerDeployer) PhaseDeploying() (ctrl.Result, error) {
	pod, err := d.ConfigurePod()
	if err != nil {
		return d.Error("Failed to get pod relevant to configure job", err)
	}

	yaml, err := k8s_utils.GetPodLog(d.Ctx, d.RestConfig, pod)
	if err != nil {
		return d.Error("Failed to get pod relevant to configure job", err)
	}

	objects, err := k8s_utils.ParseK8sObjects(yaml)
	if err != nil {
		return d.Error("Failed to parse output of installer", err)
	}

	err = k8s_utils.Create(d.Ctx, d.Client, objects, d.Log)
	if err != nil {
		return d.Error("Failed to create Falcon Container objects in the cluster", err)
	}

	return d.NextPhase(falconv1alpha1.PhaseDone)
}
