package falcon_container_deployer

import (
	imagev1 "github.com/openshift/api/image/v1"
	types "k8s.io/apimachinery/pkg/types"
)

const (
	IMAGE_STREAM_NAME = "falcon-container"
)

func (d *FalconContainerDeployer) GetImageStream() (*imagev1.ImageStream, error) {
	var stream imagev1.ImageStream
	err := d.Client.Get(d.Ctx, types.NamespacedName{Name: IMAGE_STREAM_NAME, Namespace: d.Namespace()}, &stream)
	return &stream, err
}
