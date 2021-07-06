package falcon_container_deployer

import (
	"github.com/crowdstrike/falcon-operator/pkg/k8s_utils"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
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
	return nil
}

func (d *FalconContainerDeployer) finalizeDeleteObjects() {

	pod, err := d.ConfigurePod()
	if err != nil {
		return
	}
	yaml, err := k8s_utils.GetPodLog(d.Ctx, d.RestConfig, pod)
	if err != nil {
		return
	}
	objects, err := k8s_utils.ParseK8sObjects(yaml)
	if err != nil {
		return
	}
	_ = k8s_utils.Delete(d.Ctx, d.Client, objects, d.Log)
}
