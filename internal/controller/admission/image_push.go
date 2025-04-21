package controllers

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

func (r *FalconAdmissionReconciler) PushImage(ctx context.Context, log logr.Logger, falconAdmission *falconv1alpha1.FalconAdmission) error {
	registryUri, err := r.registryUri(ctx, falconAdmission)
	if err != nil {
		return err
	}

	// If we have version locking enabled (as it is by default), use the already configured version if present
	if r.versionLock(falconAdmission) {
		return nil
	}

	pushAuth, err := r.pushAuth(ctx, falconAdmission)
	if err != nil {
		return err
	}

	log.Info("Found secret for image push", "Secret.Name", pushAuth.Name())
	image := image.NewImageRefresher(ctx, log, r.falconApiConfig(ctx, falconAdmission), pushAuth, falconAdmission.Spec.Registry.TLS.InsecureSkipVerify)
	version := falconAdmission.Spec.Version

	tag, err := image.Refresh(registryUri, falcon.KacSensor, version)
	if err != nil {
		return fmt.Errorf("Cannot push Falcon Admission Image: %v", err)
	}

	log.Info("Falcon Admission Controller Image pushed successfully", "Image.Tag", tag)
	falconAdmission.Status.Sensor = &tag

	imageUri, err := r.imageUri(ctx, falconAdmission)
	if err != nil {
		return fmt.Errorf("Cannot identify Falcon Admission Image: %v", err)
	}

	meta.SetStatusCondition(&falconAdmission.Status.Conditions, metav1.Condition{
		Type:    "ImageReady",
		Status:  metav1.ConditionTrue,
		Message: imageUri,
		Reason:  "Pushed",
	})

	return r.Client.Status().Update(ctx, falconAdmission)
}

func (r *FalconAdmissionReconciler) verifyCrowdStrike(ctx context.Context, log logr.Logger, falconAdmission *falconv1alpha1.FalconAdmission) (bool, error) {
	if _, err := r.setImageTag(ctx, falconAdmission); err != nil {
		return false, fmt.Errorf("Cannot set Falcon Registry Tag: %s", err)
	}

	imageUri, err := r.imageUri(ctx, falconAdmission)
	if err != nil {
		return false, fmt.Errorf("Cannot find Falcon Registry URI: %s", err)
	}

	condition := meta.IsStatusConditionPresentAndEqual(falconAdmission.Status.Conditions, falconv1alpha1.ConditionImageReady, metav1.ConditionTrue)
	if condition {
		return false, nil
	}

	log.Info("Skipping push of Falcon Admission image to local registry. Remote CrowdStrike registry will be used.")
	meta.SetStatusCondition(&falconAdmission.Status.Conditions, metav1.Condition{
		Status:             metav1.ConditionTrue,
		Reason:             falconv1alpha1.ReasonDiscovered,
		Message:            imageUri,
		Type:               falconv1alpha1.ConditionImageReady,
		ObservedGeneration: falconAdmission.GetGeneration(),
	})

	return true, r.Status().Update(ctx, falconAdmission)
}

func (r *FalconAdmissionReconciler) registryUri(ctx context.Context, falconAdmission *falconv1alpha1.FalconAdmission) (string, error) {
	switch falconAdmission.Spec.Registry.Type {
	case falconv1alpha1.RegistryTypeOpenshift:
		imageStream := &imagev1.ImageStream{}
		err := common.GetNamespacedObject(ctx, r.Client, r.Reader, types.NamespacedName{Name: "falcon-admission-controller", Namespace: r.imageNamespace(falconAdmission)}, imageStream)
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

		return "gcr.io/" + projectId + "/falcon-kac", nil
	case falconv1alpha1.RegistryTypeECR:
		repo, err := aws.UpsertECRRepo(ctx, "falcon-kac")
		if err != nil {
			return "", fmt.Errorf("Cannot get target docker URI for ECR repository: %v", err)
		}

		return *repo.RepositoryUri, nil
	case falconv1alpha1.RegistryTypeACR:
		if falconAdmission.Spec.Registry.AcrName == nil {
			return "", fmt.Errorf("Cannot push Falcon Image locally to ACR. acr_name was not specified")
		}

		return fmt.Sprintf("%s.azurecr.io/falcon-kac", *falconAdmission.Spec.Registry.AcrName), nil
	case falconv1alpha1.RegistryTypeCrowdStrike:
		cloud, err := falconAdmission.Spec.FalconAPI.FalconCloud(ctx)
		if err != nil {
			return "", err
		}

		return falcon.FalconContainerSensorImageURI(cloud, falcon.KacSensor), nil
	default:
		return "", fmt.Errorf("Unrecognized registry type: %s", falconAdmission.Spec.Registry.Type)
	}
}

func (r *FalconAdmissionReconciler) imageUri(ctx context.Context, falconAdmission *falconv1alpha1.FalconAdmission) (string, error) {
	if falconAdmission.Spec.Image != "" {
		return falconAdmission.Spec.Image, nil
	}

	admissionImage := os.Getenv("RELATED_IMAGE_ADMISSION_CONTROLLER")
	if admissionImage != "" && falconAdmission.Spec.FalconAPI == nil {
		return admissionImage, nil
	}

	registryUri, err := r.registryUri(ctx, falconAdmission)
	if err != nil {
		return "", err
	}

	imageTag, err := r.setImageTag(ctx, falconAdmission)
	if err != nil {
		return "", fmt.Errorf("failed to set Falcon Admission Image version: %v", err)
	}

	return fmt.Sprintf("%s:%s", registryUri, imageTag), nil
}

func (r *FalconAdmissionReconciler) getImageTag(falconAdmission *falconv1alpha1.FalconAdmission) (string, error) {
	if falconAdmission.Status.Sensor != nil && *falconAdmission.Status.Sensor != "" {
		return *falconAdmission.Status.Sensor, nil
	}

	return "", fmt.Errorf("Unable to get falcon admission container image version")
}

func (r *FalconAdmissionReconciler) setImageTag(ctx context.Context, falconAdmission *falconv1alpha1.FalconAdmission) (string, error) {
	// If version locking is enabled and a version is already set in status, return the current version
	if r.versionLock(falconAdmission) {
		if tag, err := r.getImageTag(falconAdmission); err == nil {
			return tag, err
		}
	}

	// If an Image URI is set, use it for our version
	if falconAdmission.Spec.Image != "" {
		falconAdmission.Status.Sensor = common.ImageVersion(falconAdmission.Spec.Image)

		return *falconAdmission.Status.Sensor, r.Client.Status().Update(ctx, falconAdmission)
	}

	if os.Getenv("RELATED_IMAGE_ADMISSION_CONTROLLER") != "" && falconAdmission.Spec.FalconAPI == nil {
		image := os.Getenv("RELATED_IMAGE_ADMISSION_CONTROLLER")
		falconAdmission.Status.Sensor = common.ImageVersion(image)

		return *falconAdmission.Status.Sensor, r.Client.Status().Update(ctx, falconAdmission)
	}

	// Otherwise, get the newest version matching the requested version string
	registry, err := falcon_registry.NewFalconRegistry(ctx, r.falconApiConfig(ctx, falconAdmission))
	if err != nil {
		return "", err
	}

	tag, err := registry.LastContainerTag(ctx, falcon.KacSensor, falconAdmission.Spec.Version)
	if err == nil {
		falconAdmission.Status.Sensor = common.ImageVersion(tag)
	}

	return tag, err
}

func (r *FalconAdmissionReconciler) pushAuth(ctx context.Context, falconAdmission *falconv1alpha1.FalconAdmission) (auth.Credentials, error) {
	return pushtoken.GetCredentials(ctx, falconAdmission.Spec.Registry.Type,
		k8s_utils.QuerySecretsInNamespace(r.Client, r.imageNamespace(falconAdmission)),
	)
}

func (r *FalconAdmissionReconciler) imageNamespace(falconAdmission *falconv1alpha1.FalconAdmission) string {
	if falconAdmission.Spec.Registry.Type == falconv1alpha1.RegistryTypeOpenshift {
		// Within OpenShift, ImageStreams are separated by namespaces. The "openshift" namespace
		// is shared and images pushed there can be referenced by deployments in other namespaces
		return "openshift"
	}
	return falconAdmission.Spec.InstallNamespace
}

func (r *FalconAdmissionReconciler) falconApiConfig(ctx context.Context, falconAdmission *falconv1alpha1.FalconAdmission) *falcon.ApiConfig {
	if falconAdmission.Spec.FalconAPI == nil {
		return nil
	}

	cfg := falconAdmission.Spec.FalconAPI.ApiConfig()
	cfg.Context = ctx
	return cfg
}

func (r *FalconAdmissionReconciler) imageMirroringEnabled(falconAdmission *falconv1alpha1.FalconAdmission) bool {
	return falconAdmission.Spec.Registry.Type != falconv1alpha1.RegistryTypeCrowdStrike
}

func (r *FalconAdmissionReconciler) versionLock(falconAdmission *falconv1alpha1.FalconAdmission) bool {
	return (falconAdmission.Spec.Version != nil && falconAdmission.Status.Sensor != nil && strings.Contains(*falconAdmission.Status.Sensor, *falconAdmission.Spec.Version)) || (falconAdmission.Spec.Version == nil && falconAdmission.Status.Sensor != nil)
}
