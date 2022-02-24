package falcon_container_deployer

import (
	"context"
	"time"

	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
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
	Scheme     *runtime.Scheme
}

func (d *FalconContainerDeployer) Reconcile() (ctrl.Result, error) {
	if d.isToBeDeleted() {
		if d.containsFinalizer() {
			if err := d.finalize(); err != nil {
				return ctrl.Result{}, err
			}
			d.removeFinalizer()
			return ctrl.Result{}, d.Client.Update(d.Ctx, d.Instance)
		}
		return ctrl.Result{}, nil
	}

	if !d.containsFinalizer() {
		d.addFinalizer()
		return ctrl.Result{}, d.Client.Update(d.Ctx, d.Instance)
	}

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
	case falconv1alpha1.PhaseValidating:
		return d.PhaseValidating()
	}

	return ctrl.Result{}, nil

}

func (d *FalconContainerDeployer) PhasePending() (ctrl.Result, error) {
	d.Instance.Status.SetInitialConditions()

	_, err := d.UpsertNamespace(d.Namespace())
	if err != nil {
		return d.Error("Failed to upsert Namespace", err)
	}

	switch d.Instance.Spec.Registry.Type {
	case falconv1alpha1.RegistryTypeECR:
		_, err := d.UpsertECRRepo()
		if err != nil {
			return d.Error("Failed to create ECR repository", err)
		}
	case falconv1alpha1.RegistryTypeOpenshift:
		stream, err := d.UpsertImageStream()
		if err != nil {
			return d.Error("failed to upsert Image Stream", err)
		}
		if stream == nil {
			// It takes few moment for the ImageStream to be ready (shortly after it has been created)
			return ctrl.Result{RequeueAfter: time.Second * 5}, nil
		}
	}
	return d.NextPhase(falconv1alpha1.PhaseBuilding)
}

func (d *FalconContainerDeployer) PhaseBuilding() (ctrl.Result, error) {
	if d.imageMirroringEnabled() {
		err := d.PushImage()
		if err != nil {
			return d.Error("Cannot refresh Falcon Container image", err)
		}
	} else {
		updated, err := d.verifyCrowdStrikeRegistry()
		if err != nil {
			return d.Error("Falcon Container Image not ready: ", err)
		}
		if updated {
			return ctrl.Result{}, nil
		}
		err = d.UpsertCrowdStrikeSecrets()
		if err != nil {
			return d.Error("failed to upsert falcon pulltoken secrets", err)
		}
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
	if !k8s_utils.IsPodCompleted(pod) {
		d.Log.Info("Waiting for pod completion", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
		return ctrl.Result{RequeueAfter: time.Second * 5}, nil
	}

	// (Step 6) obtain job output
	_, err = k8s_utils.GetPodLog(d.Ctx, d.RestConfig, pod)
	if err != nil {
		return d.Error("Failed to get pod relevant to configure job", err)
	}

	d.Instance.Status.SetCondition(&metav1.Condition{
		Type:   "InstallerComplete",
		Status: metav1.ConditionTrue,
		Reason: "Completed",
	})

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

	err = d.deployInjector(objects)
	if err != nil {
		return d.Error("Failed to create Falcon Container objects in the cluster", err)
	}

	return d.NextPhase(falconv1alpha1.PhaseValidating)
}

func (d *FalconContainerDeployer) PhaseValidating() (ctrl.Result, error) {
	pod, err := d.InjectorPod()
	if err != nil {
		d.Log.Info("Waiting for Falcon Container Injector to deploy", "status", err.Error())
		return ctrl.Result{RequeueAfter: time.Second * 5}, nil
	}
	if !k8s_utils.IsPodRunning(pod) {
		d.Log.Info("Waiting for Falcon Container Injector to start")
	}

	d.Instance.Status.SetCondition(&metav1.Condition{
		Type:   "Complete",
		Status: metav1.ConditionTrue,
		Reason: "Deployed",
	})

	return d.NextPhase(falconv1alpha1.PhaseDone)
}
