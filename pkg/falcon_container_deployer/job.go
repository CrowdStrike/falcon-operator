package falcon_container_deployer

import (
	batchv1 "k8s.io/api/batch/v1"
	types "k8s.io/apimachinery/pkg/types"
)

const (
	JOB_NAME = "falcon-configure"
)

func (d *FalconContainerDeployer) GetJob() (*batchv1.Job, error) {
	job := batchv1.Job{}
	err := d.Client.Get(d.Ctx, types.NamespacedName{Name: JOB_NAME, Namespace: d.Namespace()}, &job)
	return &job, err
}
