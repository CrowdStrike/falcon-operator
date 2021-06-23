package falcon_container_deployer

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crowdstrike/falcon-operator/pkg/falcon_container"
)

func (d *FalconContainerDeployer) PushImage() error {
	err := d.EnsureDockercfg()
	if err != nil {
		return fmt.Errorf("Cannot find dockercfg secret from the current namespace: %v", err)
	}

	registryUri, err := d.registryUri()
	if err != nil {
		return err
	}
	image := falcon_container.NewImageRefresher(d.Ctx, d.Log, d.Instance.Spec.FalconAPI.ApiConfig(), d.Instance.Spec.Registry.TLS.InsecureSkipVerify)
	err = image.Refresh(registryUri)
	if err != nil {
		return err
	}
	d.Log.Info("Falcon Container Image pushed successfully")
	d.Instance.Status.SetCondition(&metav1.Condition{
		Type:    "ImageReady",
		Status:  metav1.ConditionTrue,
		Message: registryUri,
		Reason:  "Pushed",
	})
	return nil
}

func (d *FalconContainerDeployer) registryUri() (string, error) {
	imageStream, err := d.GetImageStream()
	if err != nil {
		return "", err
	}
	return imageStream.Status.DockerImageRepository, nil
}
