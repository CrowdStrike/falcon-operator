package falcon

import (
	"context"
	"fmt"
	"os"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/util/retry"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/internal/controller/common/sensor"
	"github.com/crowdstrike/falcon-operator/internal/controller/image"
	"github.com/crowdstrike/falcon-operator/pkg/aws"
	"github.com/crowdstrike/falcon-operator/pkg/common"
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

func (r *FalconContainerReconciler) PushImage(ctx context.Context, log logr.Logger, falconContainer *falconv1alpha1.FalconContainer) error {
	registryUri, err := r.registryUri(ctx, falconContainer)
	if err != nil {
		return err
	}

	// If we have version locking enabled (as it is by default), use the already configured version if present
	if r.versionLock(falconContainer) {
		return nil
	}

	pushAuth, err := r.pushAuth(ctx, falconContainer)
	if err != nil {
		return err
	}

	log.Info("Found secret for image push", "Secret.Name", pushAuth.Name())
	falconApiConfig, apiConfigErr := r.falconApiConfig(ctx, falconContainer)
	if apiConfigErr != nil {
		return apiConfigErr
	}

	image := image.NewImageRefresher(ctx, log, falconApiConfig, pushAuth, falconContainer.Spec.Registry.TLS.InsecureSkipVerify)
	version := falconContainer.Spec.Version

	tag, err := image.Refresh(registryUri, falcon.SidecarSensor, version)
	if err != nil {
		return fmt.Errorf("Cannot push Falcon Container Image: %v", err)
	}

	log.Info("Falcon Container Image pushed successfully", "Image.Tag", tag)
	falconContainer.Status.Sensor = &tag

	imageUri, err := r.imageUri(ctx, falconContainer)
	if err != nil {
		return fmt.Errorf("Cannot identify Falcon Container Image: %v", err)
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		meta.SetStatusCondition(&falconContainer.Status.Conditions, metav1.Condition{
			Type:    "ImageReady",
			Status:  metav1.ConditionTrue,
			Message: imageUri,
			Reason:  "Pushed",
		})

		return r.Client.Status().Update(ctx, falconContainer)
	})

	return err
}

func (r *FalconContainerReconciler) verifyCrowdStrikeRegistry(ctx context.Context, log logr.Logger, falconContainer *falconv1alpha1.FalconContainer) (bool, error) {
	if _, err := r.setImageTag(ctx, falconContainer); err != nil {
		return false, fmt.Errorf("Cannot set Falcon Registry Tag: %s", err)
	}
	log.Info("Skipping push of Falcon Container image to local registry. Remote CrowdStrike registry will be used.")

	imageUri, err := r.imageUri(ctx, falconContainer)
	if err != nil {
		return false, fmt.Errorf("Cannot find Falcon Registry URI: %s", err)
	}

	condition := meta.IsStatusConditionPresentAndEqual(falconContainer.Status.Conditions, falconv1alpha1.ConditionImageReady, metav1.ConditionTrue)
	if condition {
		return false, nil
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		meta.SetStatusCondition(&falconContainer.Status.Conditions, metav1.Condition{
			Status:             metav1.ConditionTrue,
			Reason:             falconv1alpha1.ReasonDiscovered,
			Message:            imageUri,
			Type:               falconv1alpha1.ConditionImageReady,
			ObservedGeneration: falconContainer.GetGeneration(),
		})

		return r.Client.Status().Update(ctx, falconContainer)
	})

	return true, err
}

func (r *FalconContainerReconciler) registryUri(ctx context.Context, falconContainer *falconv1alpha1.FalconContainer) (string, error) {
	switch falconContainer.Spec.Registry.Type {
	case falconv1alpha1.RegistryTypeOpenshift:
		imageStream := &imagev1.ImageStream{}
		err := common.GetNamespacedObject(ctx, r.Client, r.Reader, types.NamespacedName{Name: imageStreamName, Namespace: r.imageNamespace(falconContainer)}, imageStream)
		if err != nil {
			return "", err
		}

		if imageStream.Status.DockerImageRepository == "" {
			return "", fmt.Errorf("Unable to find route to OpenShift on-cluster registry. Please verify that OpenShift on-cluster registry is up and running.")
		}

		return imageStream.Status.DockerImageRepository, nil
	case falconv1alpha1.RegistryTypeGCR:
		projectId, err := gcp.GetProjectID()
		if err != nil {
			return "", fmt.Errorf("Cannot get GCP Project ID: %v", err)
		}

		return "gcr.io/" + projectId + "/falcon-container", nil
	case falconv1alpha1.RegistryTypeECR:
		repo, err := aws.UpsertECRRepo(ctx, "falcon-container")
		if err != nil {
			return "", fmt.Errorf("Cannot get target docker URI for ECR repository: %v", err)
		}

		return *repo.RepositoryUri, nil
	case falconv1alpha1.RegistryTypeACR:
		if falconContainer.Spec.Registry.AcrName == nil {
			return "", fmt.Errorf("Cannot push Falcon Image locally to ACR. acr_name was not specified")
		}

		return fmt.Sprintf("%s.azurecr.io/falcon-container", *falconContainer.Spec.Registry.AcrName), nil
	case falconv1alpha1.RegistryTypeCrowdStrike:
		cloud, err := falconContainer.Spec.FalconAPI.FalconCloudWithSecret(ctx, r.Reader, falconContainer.Spec.FalconSecret)
		if err != nil {
			return "", err
		}

		return falcon.FalconContainerSensorImageURI(cloud, falcon.SidecarSensor), nil
	default:
		return "", fmt.Errorf("Unrecognized registry type: %s", falconContainer.Spec.Registry.Type)
	}
}

func (r *FalconContainerReconciler) imageUri(ctx context.Context, falconContainer *falconv1alpha1.FalconContainer) (string, error) {
	if falconContainer.Spec.Image != nil && *falconContainer.Spec.Image != "" {
		return *falconContainer.Spec.Image, nil
	}

	sidecarImage := os.Getenv("RELATED_IMAGE_SIDECAR_SENSOR")
	if sidecarImage != "" && falconContainer.Spec.FalconAPI == nil {
		return sidecarImage, nil
	}

	registryUri, err := r.registryUri(ctx, falconContainer)
	if err != nil {
		return "", err
	}

	imageTag, err := r.setImageTag(ctx, falconContainer)
	if err != nil {
		return "", fmt.Errorf("failed to set Falcon Container Image version: %v", err)
	}

	if falconContainer.Spec.Registry.Type == falconv1alpha1.RegistryTypeCrowdStrike {
		semver := strings.Split(imageTag, "-")[0]
		if !falcon_registry.IsMinimumUnifiedSensorVersion(semver, falcon.KacSensor) {
			cloud, err := falconContainer.Spec.FalconAPI.FalconCloudWithSecret(ctx, r.Reader, falconContainer.Spec.FalconSecret)
			if err != nil {
				return "", err
			}
			registryUri = falcon.FalconContainerSensorImageURI(cloud, falcon.RegionedSidecarSensor)
		}
	}

	return fmt.Sprintf("%s:%s", registryUri, imageTag), nil
}

func (r *FalconContainerReconciler) getImageTag(falconContainer *falconv1alpha1.FalconContainer) (string, error) {
	if falconContainer.Status.Sensor != nil && *falconContainer.Status.Sensor != "" {
		return *falconContainer.Status.Sensor, nil
	}

	return "", fmt.Errorf("Unable to get falcon container version")
}

func (r *FalconContainerReconciler) setImageTag(ctx context.Context, falconContainer *falconv1alpha1.FalconContainer) (string, error) {
	// If version locking is enabled and a version is already set in status, return the current version
	if r.versionLock(falconContainer) {
		if tag, err := r.getImageTag(falconContainer); err == nil {
			return tag, err
		}
	}

	// If an Image URI is set, use it for our version
	if falconContainer.Spec.Image != nil && *falconContainer.Spec.Image != "" {
		falconContainer.Status.Sensor = common.ImageVersion(*falconContainer.Spec.Image)

		return *falconContainer.Status.Sensor, r.Client.Status().Update(ctx, falconContainer)
	}

	if os.Getenv("RELATED_IMAGE_SIDECAR_SENSOR") != "" && falconContainer.Spec.FalconAPI == nil {
		image := os.Getenv("RELATED_IMAGE_SIDECAR_SENSOR")
		falconContainer.Status.Sensor = common.ImageVersion(image)

		return *falconContainer.Status.Sensor, r.Client.Status().Update(ctx, falconContainer)
	}

	falconApiConfig, apiConfigErr := r.falconApiConfig(ctx, falconContainer)
	if apiConfigErr != nil {
		return "", apiConfigErr
	}

	imageRepo, err := sensor.NewImageRepository(ctx, falconApiConfig)
	if err != nil {
		return "", err
	}

	tag, err := imageRepo.GetPreferredImage(ctx, falcon.SidecarSensor, falconContainer.Spec.Version, falconContainer.Spec.Advanced.UpdatePolicy)
	if err == nil {
		falconContainer.Status.Sensor = common.ImageVersion(tag)
	}

	return tag, err
}

func (r *FalconContainerReconciler) pushAuth(ctx context.Context, falconContainer *falconv1alpha1.FalconContainer) (auth.Credentials, error) {
	return pushtoken.GetCredentials(ctx, falconContainer.Spec.Registry.Type,
		k8s_utils.QuerySecretsInNamespace(r.Client, r.imageNamespace(falconContainer)),
	)
}

func (r *FalconContainerReconciler) imageNamespace(falconContainer *falconv1alpha1.FalconContainer) string {
	if falconContainer.Spec.Registry.Type == falconv1alpha1.RegistryTypeOpenshift {
		// Within OpenShift, ImageStreams are separated by namespaces. The "openshift" namespace
		// is shared and images pushed there can be referenced by deployments in other namespaces
		return "openshift"
	}
	return falconContainer.Spec.InstallNamespace
}

func (r *FalconContainerReconciler) falconApiConfig(
	ctx context.Context,
	falconContainer *falconv1alpha1.FalconContainer,
) (*falcon.ApiConfig, error) {
	cfg, err := falconContainer.Spec.FalconAPI.ApiConfigWithSecret(ctx, r.Reader, falconContainer.Spec.FalconSecret)
	cfg.Context = ctx
	return cfg, err
}

func (r *FalconContainerReconciler) imageMirroringEnabled(falconContainer *falconv1alpha1.FalconContainer) bool {
	return falconContainer.Spec.Registry.Type != falconv1alpha1.RegistryTypeCrowdStrike
}

func (r *FalconContainerReconciler) versionLock(falconContainer *falconv1alpha1.FalconContainer) bool {
	if falconContainer.Status.Sensor == nil || falconContainer.Spec.Advanced.HasUpdatePolicy() || falconContainer.Spec.Advanced.IsAutoUpdating() {
		return false
	}

	return falconContainer.Spec.Version == nil || strings.Contains(*falconContainer.Status.Sensor, *falconContainer.Spec.Version)
}
