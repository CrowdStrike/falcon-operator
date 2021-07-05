package falcon_container_deployer

import "sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

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
	return nil
}
