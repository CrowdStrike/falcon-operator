package falcon_container_deployer

import (
	"github.com/crowdstrike/falcon-operator/pkg/falcon_container"
)

func (d *FalconContainerDeployer) PushImage() error {
	imageStream, err := d.GetImageStream()
	if err != nil {
		return err
	}
	image := falcon_container.NewImageRefresher(d.Ctx, d.Log, d.Instance.Spec.FalconAPI.ApiConfig())
	err = image.Refresh(imageStream.Status.DockerImageRepository)
	if err != nil {
		return err
	}
	d.Log.Info("Falcon Container Image pushed successfully")
	return nil
}
