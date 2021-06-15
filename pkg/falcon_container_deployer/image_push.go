package falcon_container_deployer

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crowdstrike/falcon-operator/pkg/falcon_container"
)

func (d *FalconContainerDeployer) PushImage() error {
	imageStream, err := d.GetImageStream()
	if err != nil {
		return err
	}
	image := falcon_container.NewImageRefresher(d.Ctx, d.Log, d.Instance.Spec.FalconAPI.ApiConfig(), d.Instance.Spec.Registry.TLS.InsecureSkipVerify)
	err = image.Refresh(imageStream.Status.DockerImageRepository)
	if err != nil {
		return err
	}
	d.Log.Info("Falcon Container Image pushed successfully")
	d.Instance.Status.SetCondition(&metav1.Condition{
		Type:    "ImageReady",
		Status:  metav1.ConditionTrue,
		Message: imageStream.Status.DockerImageRepository,
		Reason:  "Pushed",
	})
	return nil
}
