package falcon_container_deployer

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/aws"
	"github.com/crowdstrike/falcon-operator/pkg/falcon_container"
	"github.com/crowdstrike/falcon-operator/pkg/falcon_container/falcon_registry"
	"github.com/crowdstrike/falcon-operator/pkg/gcp"
	"github.com/crowdstrike/falcon-operator/pkg/registry_auth"
	"github.com/crowdstrike/gofalcon/falcon"
)

func (d *FalconContainerDeployer) PushImage() error {
	registryUri, err := d.registryUri()
	if err != nil {
		return err
	}

	if d.Instance.Spec.Registry.Type == falconv1alpha1.RegistryTypeCrowdStrike {
		d.Log.Info("Skipping push of Falcon Container image to local registry. Remote CrowdStrike registry will be used.")
		d.Instance.Status.SetCondition(&metav1.Condition{
			Type:    "ImageReady",
			Status:  metav1.ConditionTrue,
			Message: registryUri,
			Reason:  "Discovered",
		})
		_, err := d.imageTag()
		return err
	}

	pushAuth, err := d.pushAuth()
	if err != nil {
		return err
	}

	d.Log.Info("Found secret for image push", "Secret.Name", pushAuth.Name())
	image := falcon_container.NewImageRefresher(d.Ctx, d.Log, d.falconApiConfig(), d.Instance.Spec.FalconAPI.CID, pushAuth, d.Instance.Spec.Registry.TLS.InsecureSkipVerify)
	falconImageTag, err := image.Refresh(registryUri, d.Instance.Spec.Version)
	if err != nil {
		return err
	}
	_ = falconImageTag
	d.Log.Info("Falcon Container Image pushed successfully")
	d.Instance.Status.Version = &falconImageTag
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
	case falconv1alpha1.RegistryTypeECR:
		repo, err := d.UpsertECRRepo()
		if err != nil {
			return "", fmt.Errorf("Cannot get target docker URI for ECR repository: %v", err)
		}
		return *repo.RepositoryUri, nil
	case falconv1alpha1.RegistryTypeACR:
		if d.Instance.Spec.Registry.AcrName == nil {
			return "", fmt.Errorf("Cannot push Falcon Image locally to ACR. acr_name was not specified")
		}
		return fmt.Sprintf("%s.azurecr.io/falcon-container", *d.Instance.Spec.Registry.AcrName), nil
	case falconv1alpha1.RegistryTypeCrowdStrike:
		return falcon_registry.ImageURI(d.Instance.Spec.FalconAPI.FalconCloud()), nil
	default:
		return "", fmt.Errorf("Unrecognized registry type: %s", d.Instance.Spec.Registry.Type)
	}
}

func (d *FalconContainerDeployer) imageUri() (string, error) {
	registryUri, err := d.registryUri()
	if err != nil {
		return "", err
	}

	imageTag, err := d.imageTag()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%s", registryUri, imageTag), nil
}

func (d *FalconContainerDeployer) imageTag() (string, error) {
	if d.Instance.Status.Version != nil && *d.Instance.Status.Version != "" {
		return *d.Instance.Status.Version, nil
	}
	registry, err := falcon_registry.NewFalconRegistry(d.falconApiConfig(), d.Instance.Spec.FalconAPI.CID, d.Log)
	if err != nil {
		return "", err
	}
	tag, _, _, err := registry.PullInfo(d.Ctx, d.Instance.Spec.Version)
	if err == nil {
		d.Instance.Status.Version = &tag
	}
	return tag, err
}

func (d *FalconContainerDeployer) pushAuth() (registry_auth.Credentials, error) {
	switch d.Instance.Spec.Registry.Type {
	case falconv1alpha1.RegistryTypeECR:
		cfg, err := aws.NewConfig()
		if err != nil {
			return nil, err
		}
		token, err := cfg.ECRLogin(d.Ctx)
		if err != nil {
			return nil, err
		}
		return registry_auth.ECRCredentials(string(token))
	default:
		namespace := d.imageNamespace()
		secrets := &corev1.SecretList{}
		err := d.Client.List(d.Ctx, secrets, client.InNamespace(namespace))
		if err != nil {
			return nil, err
		}

		creds := registry_auth.GetCredentials(secrets.Items)
		if creds == nil {
			return nil, fmt.Errorf("Cannot find suitable secret in namespace %s to push falcon-image to the registry", namespace)
		}
		return creds, nil
	}
}

func (d *FalconContainerDeployer) imageNamespace() string {
	if d.Instance.Spec.Registry.Type == falconv1alpha1.RegistryTypeOpenshift {
		// Within OpenShift, ImageStreams are separated by namespaces. The "openshift" namespace
		// is shared and images pushed there can be referenced by deployments in other namespaces
		return "openshift"
	}
	return d.Namespace()
}

func (d *FalconContainerDeployer) falconApiConfig() *falcon.ApiConfig {
	cfg := d.Instance.Spec.FalconAPI.ApiConfig()
	cfg.Context = d.Ctx
	return cfg
}
