package falcon

import (
	"context"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/gcp"
	"github.com/crowdstrike/falcon-operator/pkg/k8s_utils"
	"github.com/crowdstrike/falcon-operator/pkg/registry/auth"
	"github.com/crowdstrike/falcon-operator/pkg/registry/falcon_registry"
	"github.com/crowdstrike/falcon-operator/pkg/registry/pushtoken"
	"github.com/crowdstrike/gofalcon/falcon"
	"github.com/go-logr/logr"
	imagev1 "github.com/openshift/api/image/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	types "k8s.io/apimachinery/pkg/types"
)

func (r *FalconContainerReconciler) PushImage(ctx context.Context, log logr.Logger, falconContainer *v1alpha1.FalconContainer) error {
	registryUri, err := r.registryUri(ctx, falconContainer)
	if err != nil {
		return err
	}

	pushAuth, err := r.pushAuth(ctx, falconContainer)
	if err != nil {
		return err
	}

	log.Info("Found secret for image push", "Secret.Name", pushAuth.Name())
	image := NewImageRefresher(ctx, log, r.falconApiConfig(ctx, falconContainer), pushAuth, falconContainer.Spec.Registry.TLS.InsecureSkipVerify)
	version := falconContainer.Spec.Version

	// If we have version locking enabled (as it is by default), use the already configured version if present
	if falconContainer.Spec.VersionLocking && falconContainer.Status.Version != nil && *falconContainer.Status.Version != "" {
		return nil
	}

	tag, err := image.Refresh(registryUri, version)
	if err != nil {
		return fmt.Errorf("Cannot push Falcon Container Image: %v", err)
	}

	log.Info("Falcon Container Image pushed successfully", "Image.Tag", tag)
	falconContainer.Status.Version = &tag

	imageUri, err := r.imageUri(ctx, falconContainer)
	if err != nil {
		return fmt.Errorf("Cannot identify Falcon Container Image: %v", err)
	}

	meta.SetStatusCondition(&falconContainer.Status.Conditions, metav1.Condition{
		Type:    "ImageReady",
		Status:  metav1.ConditionTrue,
		Message: imageUri,
		Reason:  "Pushed",
	})

	return r.Client.Status().Update(ctx, falconContainer)
}

func (r *FalconContainerReconciler) verifyCrowdStrikeRegistry(ctx context.Context, log logr.Logger, falconContainer *v1alpha1.FalconContainer) (bool, error) {
	if _, err := r.setImageTag(ctx, falconContainer); err != nil {
		return false, fmt.Errorf("Cannot set Falcon Registry Tag: %s", err)
	}
	log.Info("Skipping push of Falcon Container image to local registry. Remote CrowdStrike registry will be used.")

	imageUri, err := r.imageUri(ctx, falconContainer)
	if err != nil {
		return false, fmt.Errorf("Cannot find Falcon Registry URI: %s", err)
	}

	condition := meta.IsStatusConditionPresentAndEqual(falconContainer.Status.Conditions, v1alpha1.ConditionImageReady, metav1.ConditionTrue)
	if condition {
		return false, nil
	}

	meta.SetStatusCondition(&falconContainer.Status.Conditions, metav1.Condition{
		Status:             metav1.ConditionTrue,
		Reason:             v1alpha1.ReasonDiscovered,
		Message:            imageUri,
		Type:               v1alpha1.ConditionImageReady,
		ObservedGeneration: falconContainer.GetGeneration(),
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
			return "", fmt.Errorf("Unable to find route to OpenShift on-cluster registry. Please verify that OpenShift on-cluster registry is up and running.")
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
	if falconContainer.Spec.Image != nil && *falconContainer.Spec.Image != "" {
		return *falconContainer.Spec.Image, nil
	}

	registryUri, err := r.registryUri(ctx, falconContainer)
	if err != nil {
		return "", err
	}

	imageTag, err := r.getImageTag(ctx, falconContainer)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s:%s", registryUri, imageTag), nil
}

func (r *FalconContainerReconciler) getImageTag(ctx context.Context, falconContainer *v1alpha1.FalconContainer) (string, error) {
	if falconContainer.Status.Version != nil && *falconContainer.Status.Version != "" {
		return *falconContainer.Status.Version, nil
	}

	return "", fmt.Errorf("Unable to get falcon container version")
}

func (r *FalconContainerReconciler) setImageTag(ctx context.Context, falconContainer *v1alpha1.FalconContainer) (string, error) {
	// If version locking is enabled and a version is already set in status, return the current version
	if falconContainer.Spec.VersionLocking {
		if tag, err := r.getImageTag(ctx, falconContainer); err == nil {
			return tag, err
		}
	}

	// If an Image URI is set, use it for our version
	if falconContainer.Spec.Image != nil && *falconContainer.Spec.Image != "" {
		falconContainer.Status.Version = &strings.Split(*falconContainer.Spec.Image, ":")[1]

		return *falconContainer.Status.Version, r.Client.Status().Update(ctx, falconContainer)
	} else {
		// Otherwise, get the newest version matching the requested version string
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
