package image

import (
	"context"
	"fmt"
	"os"
	"strings"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/crowdstrike/falcon-operator/pkg/registry/falcon_registry"
	"github.com/crowdstrike/gofalcon/falcon"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Resolver is the minimal interface required to resolve and store the sensor image.
type Resolver interface {
	GetK8sReader() client.Reader
	Status() client.StatusWriter
}

// Config holds the spec fields needed to resolve the sensor image URI.
type Config struct {
	Image              string
	FalconAPI          *falconv1alpha1.FalconAPI
	FalconSecret       falconv1alpha1.FalconSecret
	Version            *string
	RelatedImageEnvVar string
	BuildStage         string
	// CrowdStrikeRegistry is true when the image is pulled from the CrowdStrike registry (registry.type: crowdstrike).
	CrowdStrikeRegistry bool
}

// Discover reports whether the image is discovered from the CrowdStrike registry: image is not set, the registry type
// is crowdstrike, and Falcon API credentials are configured. Credentials alone are not enough, since they are also
// required at runtime (e.g. by the Image Analyzer) when the image is hosted in a private registry.
func Discover(cfg Config) bool {
	return cfg.Image == "" && cfg.CrowdStrikeRegistry && cfg.FalconAPI.IsConfigured(cfg.FalconSecret)
}

// lastTag queries the CrowdStrike registry for the newest Cluster Guard tag. It is a variable so tests can stub the registry.
var lastTag = func(ctx context.Context, apiConfig *falcon.ApiConfig, buildStage string, version *string) (string, error) {
	registry, err := falcon_registry.NewFalconRegistry(ctx, apiConfig)
	if err != nil {
		return "", err
	}
	return registry.LastClusterGuardTag(ctx, buildStage, version)
}

// RelatedImage returns the operator-provided image from cfg.RelatedImageEnvVar when the image is not discovered
// from the CrowdStrike registry.
func RelatedImage(cfg Config) string {
	if Discover(cfg) {
		return ""
	}
	return os.Getenv(cfg.RelatedImageEnvVar)
}

func errImageRequired(cfg Config) error {
	return fmt.Errorf("spec.image must be set: the image is only discovered when registry.type is crowdstrike and Falcon API credentials are configured, and no operator-provided image is set in %s", cfg.RelatedImageEnvVar)
}

func URI(ctx context.Context, r Resolver, cfg Config, status *falconv1alpha1.FalconCRStatus, obj client.Object) (string, error) {
	if cfg.Image != "" {
		if _, err := SetTag(ctx, r, cfg, status, obj); err != nil {
			return "", err
		}
		return cfg.Image, nil
	}

	if relatedImage := RelatedImage(cfg); relatedImage != "" {
		if _, err := SetTag(ctx, r, cfg, status, obj); err != nil {
			return "", err
		}
		return relatedImage, nil
	}

	if !Discover(cfg) {
		return "", errImageRequired(cfg)
	}

	imageTag, err := SetTag(ctx, r, cfg, status, obj)
	if err != nil {
		return "", fmt.Errorf("failed to set Falcon Cluster Guard Image version: %v", err)
	}

	cloud, err := cfg.FalconAPI.FalconCloudWithSecret(ctx, r.GetK8sReader(), cfg.FalconSecret)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s:%s", falcon_registry.ClusterGuardImageURI(cloud, cfg.BuildStage), imageTag), nil
}

func SetTag(ctx context.Context, r Resolver, cfg Config, status *falconv1alpha1.FalconCRStatus, obj client.Object) (string, error) {
	// If version locking is enabled and a version is already set in status, return the current version
	if versionLock(cfg, status) {
		if tag, err := getTag(status); err == nil {
			return tag, err
		}
	}

	if cfg.Image != "" {
		status.Sensor = common.ImageVersion(cfg.Image)
		return *status.Sensor, r.Status().Update(ctx, obj)
	}

	if img := RelatedImage(cfg); img != "" {
		log.FromContext(ctx).Info("spec.image not set and image discovery not enabled; using operator-provided image from "+cfg.RelatedImageEnvVar,
			"image", img)
		status.Sensor = common.ImageVersion(img)
		return *status.Sensor, r.Status().Update(ctx, obj)
	}

	if !Discover(cfg) {
		return "", errImageRequired(cfg)
	}

	// Otherwise, get the newest version matching the requested version string
	apiConfig, err := apiConfigFor(ctx, r, cfg)
	if err != nil {
		return "", err
	}

	tag, err := lastTag(ctx, apiConfig, cfg.BuildStage, cfg.Version)
	if err == nil {
		status.Sensor = common.ImageVersion(tag)
	}

	return tag, err
}

// Repository returns imageURI without its tag.
func Repository(imageURI string) string {
	if i := strings.LastIndex(imageURI, ":"); i > strings.LastIndex(imageURI, "/") {
		return imageURI[:i]
	}
	return imageURI
}

func getTag(status *falconv1alpha1.FalconCRStatus) (string, error) {
	if status.Sensor != nil && *status.Sensor != "" {
		return *status.Sensor, nil
	}
	return "", fmt.Errorf("unable to get Falcon Cluster Guard container image version")
}

func apiConfigFor(ctx context.Context, r Resolver, cfg Config) (*falcon.ApiConfig, error) {
	apiCfg, err := cfg.FalconAPI.ApiConfigWithSecret(ctx, r.GetK8sReader(), cfg.FalconSecret)
	apiCfg.Context = ctx
	return apiCfg, err
}

func versionLock(cfg Config, status *falconv1alpha1.FalconCRStatus) bool {
	return (cfg.Version != nil && status.Sensor != nil && strings.Contains(*status.Sensor, *cfg.Version)) || (cfg.Version == nil && status.Sensor != nil)
}
