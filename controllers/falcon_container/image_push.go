package falcon

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/gcp"
	"github.com/crowdstrike/falcon-operator/pkg/k8s_utils"
	"github.com/crowdstrike/falcon-operator/pkg/registry/auth"
	"github.com/crowdstrike/falcon-operator/pkg/registry/falcon_registry"
	"github.com/crowdstrike/falcon-operator/pkg/registry/pushtoken"
	"github.com/crowdstrike/gofalcon/falcon"
	imagev1 "github.com/openshift/api/image/v1"
	types "k8s.io/apimachinery/pkg/types"
)

func (r *FalconContainerReconciler) PushImage(ctx context.Context, falconContainer *v1alpha1.FalconContainer) error {
	registryUri, err := r.registryUri(ctx, falconContainer)
	if err != nil {
		return err
	}

	pushAuth, err := r.pushAuth(ctx, falconContainer)
	if err != nil {
		return err
	}

	r.Log.Info("Found secret for image push", "Secret.Name", pushAuth.Name())
	image := NewImageRefresher(ctx, r.Log, r.falconApiConfig(ctx, falconContainer), pushAuth, falconContainer.Spec.Registry.TLS.InsecureSkipVerify)
	falconImageTag, err := image.Refresh(registryUri, falconContainer.Spec.Version)
	if err != nil {
		return err
	}
	_ = falconImageTag
	r.Log.Info("Falcon Container Image pushed successfully")
	falconContainer.Status.Version = &falconImageTag
	falconContainer.Status.SetCondition(&metav1.Condition{
		Type:    "ImageReady",
		Status:  metav1.ConditionTrue,
		Message: registryUri,
		Reason:  "Pushed",
	})
	return r.Client.Status().Update(ctx, falconContainer)
}

func (r *FalconContainerReconciler) verifyCrowdStrikeRegistry(ctx context.Context, falconContainer *v1alpha1.FalconContainer) (bool, error) {
	conditionName := "ImageReady"
	condition := falconContainer.Status.GetCondition(conditionName)
	if condition != nil && condition.Status == metav1.ConditionTrue {
		return false, nil
	}

	r.Log.Info("Skipping push of Falcon Container image to local registry. Remote CrowdStrike registry will be user.")
	registryUri, err := r.registryUri(ctx, falconContainer)
	if err != nil {
		return false, err
	}
	_, err = r.imageTag(ctx, falconContainer)
	if err != nil {
		return false, fmt.Errorf("Cannot find Falcon Registry Tag: %s", err)
	}

	falconContainer.Status.SetCondition(&metav1.Condition{
		Type:    "ImageReady",
		Status:  metav1.ConditionTrue,
		Message: registryUri,
		Reason:  "Discovered",
	})

	return true, r.Client.Status().Update(ctx, falconContainer)
}

func (r *FalconContainerReconciler) registryUri(ctx context.Context, falconContainer *v1alpha1.FalconContainer) (string, error) {
	switch falconContainer.Spec.Registry.Type {
	case v1alpha1.RegistryTypeOpenshift:
		imageStream := &imagev1.ImageStream{}
		err := r.Client.Get(ctx, types.NamespacedName{Name: imageStreamName, Namespace: r.imageNamespace(falconContainer)}, imageStream)

		if err != nil {
			return "", err
		}
		if imageStream.Status.DockerImageRepository == "" {
			return "", fmt.Errorf("Unable to find route to OpenShift on-cluster registry. Please verify that OpenShift on-cluster registry is up and running")
		}

		return imageStream.Status.DockerImageRepository, nil
	case v1alpha1.RegistryTypeGCR:
		projectId, err := gcp.GetProjectID()
		if err != nil {
			return "", fmt.Errorf("Cannot get GCP Project ID: %v", err)
		}

		return "gcr.io/" + projectId + "/falcon-container", nil
	case v1alpha1.RegistryTypeECR:
		repo, err := r.UpsertECRRepo(ctx)
		if err != nil {
			return "", fmt.Errorf("Cannot get target docker URI for ECR repository: %v", err)
		}
		return *repo.RepositoryUri, nil
	case v1alpha1.RegistryTypeACR:
		if falconContainer.Spec.Registry.AcrName == nil {
			return "", fmt.Errorf("Cannot push Falcon Image locally to ACR. acr_name was not specified")
		}
		return fmt.Sprintf("%s.azurecr.io/falcon-container", *falconContainer.Spec.Registry.AcrName), nil
	case v1alpha1.RegistryTypeCrowdStrike:
		cloud, err := falconContainer.Spec.FalconAPI.FalconCloud(ctx)
		if err != nil {
			return "", err
		}
		return falcon_registry.ImageURIContainer(cloud), nil
	default:
		return "", fmt.Errorf("Unrecognized registry type: %s", falconContainer.Spec.Registry.Type)
	}
}

func (r *FalconContainerReconciler) imageUri(ctx context.Context, falconContainer *v1alpha1.FalconContainer) (string, error) {
	registryUri, err := r.registryUri(ctx, falconContainer)
	if err != nil {
		return "", err
	}

	imageTag, err := r.imageTag(ctx, falconContainer)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s:%s", registryUri, imageTag), nil
}

func (r *FalconContainerReconciler) imageTag(ctx context.Context, falconContainer *v1alpha1.FalconContainer) (string, error) {
	if falconContainer.Status.Version != nil && *falconContainer.Status.Version != "" {
		return *falconContainer.Status.Version, nil
	}
	registry, err := falcon_registry.NewFalconRegistry(ctx, r.falconApiConfig(ctx, falconContainer))
	if err != nil {
		return "", err
	}
	tag, err := registry.LastContainerTag(ctx, falconContainer.Spec.Version)
	if err == nil {
		falconContainer.Status.Version = &tag
	}
	return tag, err
}

func (r *FalconContainerReconciler) pushAuth(ctx context.Context, falconContainer *v1alpha1.FalconContainer) (auth.Credentials, error) {
	return pushtoken.GetCredentials(ctx, falconContainer.Spec.Registry.Type,
		k8s_utils.QuerySecretsInNamespace(r.Client, r.imageNamespace(falconContainer)),
	)
}

func (r *FalconContainerReconciler) imageNamespace(falconContainer *v1alpha1.FalconContainer) string {
	if falconContainer.Spec.Registry.Type == v1alpha1.RegistryTypeOpenshift {
		// Within OpenShift, ImageStreams are separated by namespaces. The "openshift" namespace
		// is shared and images pushed there can be referenced by deployments in other namespaces
		return "openshift"
	}
	return r.Namespace()
}

func (r *FalconContainerReconciler) falconApiConfig(ctx context.Context, falconContainer *v1alpha1.FalconContainer) *falcon.ApiConfig {
	cfg := falconContainer.Spec.FalconAPI.ApiConfig()
	cfg.Context = ctx
	return cfg
}

func (r *FalconContainerReconciler) imageMirroringEnabled(falconContainer *v1alpha1.FalconContainer) bool {
	return falconContainer.Spec.Registry.Type != v1alpha1.RegistryTypeCrowdStrike
}
