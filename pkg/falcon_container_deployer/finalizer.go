package falcon_container_deployer

import (
	"github.com/crowdstrike/falcon-operator/pkg/k8s_utils"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
)

const falconContainerFinalizer = "falcon.crowdstrike.com/finalizer"

func (d *FalconContainerDeployer) isToBeDeleted() bool {
	return d.Instance.GetDeletionTimestamp() != nil
}

func (d *FalconContainerDeployer) containsFinalizer() bool {
	return controllerutil.ContainsFinalizer(d.Instance, falconContainerFinalizer)
}

func (d *FalconContainerDeployer) addFinalizer() {
	controllerutil.AddFinalizer(d.Instance, falconContainerFinalizer)
}

func (d *FalconContainerDeployer) removeFinalizer() {
	controllerutil.RemoveFinalizer(d.Instance, falconContainerFinalizer)
}

func (d *FalconContainerDeployer) finalize() error {
	d.Log.Info("Running Falcon Container Finalizer")

	d.finalizeDeleteObjects()
	d.finalizeDeleteJob()

	switch d.Instance.Spec.Registry.Type {
	case falconv1alpha1.RegistryTypeOpenshift:
		stream, err := d.GetImageStream()
		if err != nil {
			d.Log.Error(err, "Could not find ImageStream for deletion")
		}
		err = d.DeleteImageStream(stream)
		if err != nil {
			d.Log.Error(err, "Could not delete ImageStream")
		}
	}

	return nil
}

func (d *FalconContainerDeployer) finalizeDeleteObjects() {
	pod, err := d.ConfigurePod()
	if err != nil {
		d.Log.Error(err, "Could not find Falcon Container Installer pod for deletion")
		return
	}
	yaml, err := k8s_utils.GetPodLog(d.Ctx, d.RestConfig, pod)
	if err != nil {
		d.Log.Error(err, "Could not fetch logs of Falcon Container Installer")
		return
	}
	objects, err := k8s_utils.ParseK8sObjects(yaml)
	if err != nil {
		d.Log.Error(err, "Could not parse Falcon Container Installer output")
		return
	}
	err = k8s_utils.Delete(d.Ctx, d.Client, objects, d.Log)
	if err != nil {
		d.Log.Error(err, "Could not delete Falcon Container from the cluster")
	}
}

func (d *FalconContainerDeployer) finalizeDeleteJob() {
	job, err := d.GetJob()
	if err != nil {
		d.Log.Error(err, "Could not get Falcon Container Installer job")
	}
	err = d.DeleteJob(job)
	if err != nil {
		d.Log.Error(err, "Cloud not delete Falcon Container Installer job")
	}
}
