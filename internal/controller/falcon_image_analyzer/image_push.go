package falcon

import (
	"context"
	"fmt"
	"os"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
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
)

func (r *FalconImageAnalyzerReconciler) PushImage(ctx context.Context, log logr.Logger, falconImageAnalyzer *falconv1alpha1.FalconImageAnalyzer) error {
	registryUri, err := r.registryUri(ctx, falconImageAnalyzer)
	if err != nil {
		return err
	}

	// If we have version locking enabled (as it is by default), use the already configured version if present
	if r.versionLock(falconImageAnalyzer) {
		return nil
	}

	pushAuth, err := r.pushAuth(ctx, falconImageAnalyzer)
	if err != nil {
		return err
	}

	log.Info("Found secret for image push", "Secret.Name", pushAuth.Name())
	falconApiConfig, err := r.falconApiConfig(ctx, falconImageAnalyzer)
	if err != nil {
		return err
	}

	image := image.NewImageRefresher(ctx, log, falconApiConfig, pushAuth, falconImageAnalyzer.Spec.Registry.TLS.InsecureSkipVerify)
	version := falconImageAnalyzer.Spec.Version

	tag, err := image.Refresh(registryUri, falcon.ImageSensor, version)
	if err != nil {
		return fmt.Errorf("Cannot push Falcon Image Analyzer Image: %v", err)
	}

	log.Info("Falcon Image Analyzer Controller Image pushed successfully", "Image.Tag", tag)
	falconImageAnalyzer.Status.Sensor = &tag

	imageUri, err := r.imageUri(ctx, falconImageAnalyzer)
	if err != nil {
		return fmt.Errorf("Cannot identify Falcon Image Analyzer Image: %v", err)
	}

	meta.SetStatusCondition(&falconImageAnalyzer.Status.Conditions, metav1.Condition{
		Type:    "ImageReady",
		Status:  metav1.ConditionTrue,
		Message: imageUri,
		Reason:  "Pushed",
	})

	return r.Client.Status().Update(ctx, falconImageAnalyzer)
}

func (r *FalconImageAnalyzerReconciler) verifyCrowdStrike(ctx context.Context, log logr.Logger, falconImageAnalyzer *falconv1alpha1.FalconImageAnalyzer) (bool, error) {
	if _, err := r.setImageTag(ctx, falconImageAnalyzer); err != nil {
		return false, fmt.Errorf("Cannot set Falcon Registry Tag: %s", err)
	}

	imageUri, err := r.imageUri(ctx, falconImageAnalyzer)
	if err != nil {
		return false, fmt.Errorf("Cannot find Falcon Registry URI: %s", err)
	}

	condition := meta.IsStatusConditionPresentAndEqual(falconImageAnalyzer.Status.Conditions, falconv1alpha1.ConditionImageReady, metav1.ConditionTrue)
	if condition {
		return false, nil
	}

	log.Info("Skipping push of Falcon Image Analyzer image to local registry. Remote CrowdStrike registry will be used.")
	meta.SetStatusCondition(&falconImageAnalyzer.Status.Conditions, metav1.Condition{
		Status:             metav1.ConditionTrue,
		Reason:             falconv1alpha1.ReasonDiscovered,
		Message:            imageUri,
		Type:               falconv1alpha1.ConditionImageReady,
		ObservedGeneration: falconImageAnalyzer.GetGeneration(),
	})

	return true, r.Status().Update(ctx, falconImageAnalyzer)
}

func (r *FalconImageAnalyzerReconciler) registryUri(ctx context.Context, falconImageAnalyzer *falconv1alpha1.FalconImageAnalyzer) (string, error) {
	switch falconImageAnalyzer.Spec.Registry.Type {
	case falconv1alpha1.RegistryTypeOpenshift:
		imageStream := &imagev1.ImageStream{}
		err := common.GetNamespacedObject(ctx, r.Client, r.Reader, types.NamespacedName{Name: "falcon-image-analyzer", Namespace: r.imageNamespace(falconImageAnalyzer)}, imageStream)
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

		return "gcr.io/" + projectId + "/falcon-imageanalyzer", nil
	case falconv1alpha1.RegistryTypeECR:
		repo, err := aws.UpsertECRRepo(ctx, "falcon-image-analyzer")
		if err != nil {
			return "", fmt.Errorf("Cannot get target docker URI for ECR repository: %v", err)
		}

		return *repo.RepositoryUri, nil
	case falconv1alpha1.RegistryTypeACR:
		if falconImageAnalyzer.Spec.Registry.AcrName == nil {
			return "", fmt.Errorf("Cannot push Falcon Image locally to ACR. acr_name was not specified")
		}

		return fmt.Sprintf("%s.azurecr.io/falcon-imageanalyzer", *falconImageAnalyzer.Spec.Registry.AcrName), nil
	case falconv1alpha1.RegistryTypeCrowdStrike:
		cloud, err := falconImageAnalyzer.Spec.FalconAPI.FalconCloudWithSecret(ctx, r.Reader, falconImageAnalyzer.Spec.FalconSecret)
		if err != nil {
			return "", err
		}

		return falcon.FalconContainerSensorImageURI(cloud, falcon.ImageSensor), nil
	default:
		return "", fmt.Errorf("Unrecognized registry type: %s", falconImageAnalyzer.Spec.Registry.Type)
	}
}

func (r *FalconImageAnalyzerReconciler) imageUri(ctx context.Context, falconImageAnalyzer *falconv1alpha1.FalconImageAnalyzer) (string, error) {
	if falconImageAnalyzer.Spec.Image != "" {
		return falconImageAnalyzer.Spec.Image, nil
	}

	imageAnalyzerImage := os.Getenv("RELATED_IMAGE_IMAGE_ANALYZER")
	if imageAnalyzerImage != "" && falconImageAnalyzer.Spec.FalconAPI == nil {
		return imageAnalyzerImage, nil
	}

	registryUri, err := r.registryUri(ctx, falconImageAnalyzer)
	if err != nil {
		return "", err
	}

	imageTag, err := r.setImageTag(ctx, falconImageAnalyzer)
	if err != nil {
		return "", fmt.Errorf("failed to set Falcon Image Analyzer Image version: %v", err)
	}

	return fmt.Sprintf("%s:%s", registryUri, imageTag), nil
}

func (r *FalconImageAnalyzerReconciler) getImageTag(falconImageAnalyzer *falconv1alpha1.FalconImageAnalyzer) (string, error) {
	if falconImageAnalyzer.Status.Sensor != nil && *falconImageAnalyzer.Status.Sensor != "" {
		return *falconImageAnalyzer.Status.Sensor, nil
	}

	return "", fmt.Errorf("Unable to get falcon image analyzer container image version")
}

func (r *FalconImageAnalyzerReconciler) setImageTag(ctx context.Context, falconImageAnalyzer *falconv1alpha1.FalconImageAnalyzer) (string, error) {
	// If version locking is enabled and a version is already set in status, return the current version
	if r.versionLock(falconImageAnalyzer) {
		if tag, err := r.getImageTag(falconImageAnalyzer); err == nil {
			return tag, err
		}
	}

	// If an Image URI is set, use it for our version
	if falconImageAnalyzer.Spec.Image != "" {
		falconImageAnalyzer.Status.Sensor = common.ImageVersion(falconImageAnalyzer.Spec.Image)

		return *falconImageAnalyzer.Status.Sensor, r.Client.Status().Update(ctx, falconImageAnalyzer)
	}

	if os.Getenv("RELATED_IMAGE_IMAGE_ANALYZER") != "" && falconImageAnalyzer.Spec.FalconAPI == nil {
		image := os.Getenv("RELATED_IMAGE_IMAGE_ANALYZER")
		falconImageAnalyzer.Status.Sensor = common.ImageVersion(image)

		return *falconImageAnalyzer.Status.Sensor, r.Client.Status().Update(ctx, falconImageAnalyzer)
	}

	// Otherwise, get the newest version matching the requested version string
	falconApiConfig, err := r.falconApiConfig(ctx, falconImageAnalyzer)
	if err != nil {
		return "", err
	}

	registry, err := falcon_registry.NewFalconRegistry(ctx, falconApiConfig)
	if err != nil {
		return "", err
	}

	tag, err := registry.LastContainerTag(ctx, falcon.ImageSensor, falconImageAnalyzer.Spec.Version)
	if err == nil {
		falconImageAnalyzer.Status.Sensor = common.ImageVersion(tag)
	}

	return tag, err
}

func (r *FalconImageAnalyzerReconciler) pushAuth(ctx context.Context, falconImageAnalyzer *falconv1alpha1.FalconImageAnalyzer) (auth.Credentials, error) {
	return pushtoken.GetCredentials(ctx, falconImageAnalyzer.Spec.Registry.Type,
		k8s_utils.QuerySecretsInNamespace(r.Client, r.imageNamespace(falconImageAnalyzer)),
	)
}

func (r *FalconImageAnalyzerReconciler) imageNamespace(falconImageAnalyzer *falconv1alpha1.FalconImageAnalyzer) string {
	if falconImageAnalyzer.Spec.Registry.Type == falconv1alpha1.RegistryTypeOpenshift {
		// Within OpenShift, ImageStreams are separated by namespaces. The "openshift" namespace
		// is shared and images pushed there can be referenced by deployments in other namespaces
		return "openshift"
	}
	return falconImageAnalyzer.Spec.InstallNamespace
}

func (r *FalconImageAnalyzerReconciler) falconApiConfig(
	ctx context.Context,
	falconImageAnalyzer *falconv1alpha1.FalconImageAnalyzer,
) (*falcon.ApiConfig, error) {
	cfg, err := falconImageAnalyzer.Spec.FalconAPI.ApiConfigWithSecret(ctx, r.Reader, falconImageAnalyzer.Spec.FalconSecret)
	cfg.Context = ctx
	return cfg, err
}

func (r *FalconImageAnalyzerReconciler) imageMirroringEnabled(falconImageAnalyzer *falconv1alpha1.FalconImageAnalyzer) bool {
	return falconImageAnalyzer.Spec.Registry.Type != falconv1alpha1.RegistryTypeCrowdStrike
}

func (r *FalconImageAnalyzerReconciler) versionLock(falconImageAnalyzer *falconv1alpha1.FalconImageAnalyzer) bool {
	return (falconImageAnalyzer.Spec.Version != nil && falconImageAnalyzer.Status.Sensor != nil && strings.Contains(*falconImageAnalyzer.Status.Sensor, *falconImageAnalyzer.Spec.Version)) || (falconImageAnalyzer.Spec.Version == nil && falconImageAnalyzer.Status.Sensor != nil)
}
