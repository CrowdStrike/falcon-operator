package falcon_container_deployer

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/falcon_container"
	"github.com/crowdstrike/falcon-operator/pkg/falcon_container/push_auth"
	"github.com/crowdstrike/falcon-operator/pkg/gcp"
)

func (d *FalconContainerDeployer) PushImage() error {
	pushAuth, err := d.pushAuth()
	if err != nil {
		return err
	}

	d.Log.Info("Found secret for image push", "Secret.Name", pushAuth.Name())
	registryUri, err := d.registryUri()
	if err != nil {
		return err
	}
	image := falcon_container.NewImageRefresher(d.Ctx, d.Log, d.Instance.Spec.FalconAPI.ApiConfig(), pushAuth, d.Instance.Spec.Registry.TLS.InsecureSkipVerify)
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
	switch d.Instance.Spec.Registry.Type {
	case falconv1alpha1.RegistryTypeOpenshift:
		imageStream, err := d.GetImageStream()

		if err != nil {
			return "", err
		}
		return imageStream.Status.DockerImageRepository, nil
	case falconv1alpha1.RegistryTypeGCR:
		projectId, err := gcp.GetProjectID()
		if err != nil {
			return "", fmt.Errorf("Cannot get GCP Project ID: %v", err)
		}

		return "gcr.io/" + projectId + "/falcon-container", nil
	default:
		return "", fmt.Errorf("Unrecognized registry type: %s", d.Instance.Spec.Registry.Type)
	}
}

func (d *FalconContainerDeployer) pushAuth() (push_auth.Credentials, error) {
	namespace := d.Namespace()
	secrets := &corev1.SecretList{}
	err := d.Client.List(d.Ctx, secrets, client.InNamespace(namespace))
	if err != nil {
		return nil, err
	}

	creds := push_auth.GetCredentials(secrets.Items)
	if creds == nil {
		return nil, fmt.Errorf("Cannot find suitable secret in namespace %s to push falcon-image to the registry", namespace)
	}
	return creds, nil
}
