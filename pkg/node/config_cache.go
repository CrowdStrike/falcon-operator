package node

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"unicode"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/internal/controller/common/sensor"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/crowdstrike/falcon-operator/pkg/falcon_api"
	"github.com/crowdstrike/falcon-operator/pkg/registry/falcon_registry"
	"github.com/crowdstrike/falcon-operator/pkg/registry/pulltoken"
	"github.com/crowdstrike/gofalcon/falcon"
	"github.com/go-logr/logr"
)

var ErrFalconAPINotConfigured = errors.New("missing falcon_api configuration")

// ConfigCache holds config values for node sensor. Those values are either provided by user or fetched dynamically. That happens transparently to the caller.
type ConfigCache struct {
	cid             string
	imageUri        string
	nodesensor      *falconv1alpha1.FalconNodeSensor
	falconApiConfig *falcon.ApiConfig
}

func NewConfigCache(ctx context.Context, nodesensor *falconv1alpha1.FalconNodeSensor) (*ConfigCache, error) {
	var err error
	cache := ConfigCache{
		nodesensor: nodesensor,
	}

	if nodesensor.Spec.FalconAPI != nil {
		cache.falconApiConfig = nodesensor.Spec.FalconAPI.ApiConfig()

		if nodesensor.Spec.FalconAPI.CID != nil {
			cache.cid = *nodesensor.Spec.FalconAPI.CID
		}
	}

	if cache.cid == "" {
		cache.cid, err = falcon_api.FalconCID(ctx, nodesensor.Spec.Falcon.CID, cache.falconApiConfig)
		if err != nil {
			return nil, err
		}
	}

	return &cache, nil
}

func (cc *ConfigCache) CID() string {
	return cc.cid
}

func (cc *ConfigCache) UsingCrowdStrikeRegistry() bool {
	if cc.nodesensor.Spec.Node.Image == "" && cc.falconApiConfig == nil {
		return os.Getenv("RELATED_IMAGE_NODE_SENSOR") == ""
	}
	return cc.nodesensor.Spec.Node.Image == ""
}

func (cc *ConfigCache) GetImageURI(ctx context.Context, logger logr.Logger) (string, error) {
	var err error
	if cc.imageUri == "" {
		cc.imageUri, err = cc.getFalconImage(ctx, cc.nodesensor)
		if err == nil {
			logger.Info("Identified Falcon Node Image", "reference", cc.imageUri)
		}
	}
	return cc.imageUri, err
}

func (cc *ConfigCache) GetPullToken(ctx context.Context) ([]byte, error) {
	if cc.falconApiConfig == nil {
		return nil, ErrFalconAPINotConfigured
	}
	return pulltoken.CrowdStrike(ctx, cc.falconApiConfig)
}

func (cc *ConfigCache) SensorEnvVars() map[string]string {
	sensorConfig := common.MakeSensorEnvMap(cc.nodesensor.Spec.Falcon.FalconSensor)
	if cc.cid != "" {
		sensorConfig["FALCONCTL_OPT_CID"] = cc.cid
	}
	if cc.nodesensor.Spec.Node.Backend != "" {
		sensorConfig["FALCONCTL_OPT_BACKEND"] = cc.nodesensor.Spec.Node.Backend
	}
	if cc.nodesensor.Spec.Falcon.Cloud != "" {
		sensorConfig["FALCONCTL_OPT_CLOUD"] = cc.nodesensor.Spec.Falcon.Cloud
	}
	if cc.nodesensor.Spec.Node.ClusterName != nil {
		sensorConfig["FALCON_CLUSTER_NAME"] = sanitizeClusterName(cc.nodesensor.Spec.Node.ClusterName)
	}

	return sensorConfig
}

func sanitizeClusterName(clusterName *string) string {
	if clusterName == nil {
		return ""
	}

	if !isClusterNameValid(*clusterName) {
		return ""
	}

	return *clusterName
}

// isClusterNameValid validates the clusterName.
// Those rules had been taken from EKS (Amazon AWS).
// See more at: https://docs.aws.amazon.com/eks/latest/APIReference/API_CreateCluster.html#API_CreateCluster_RequestSyntax
func isClusterNameValid(clusterName string) bool {
	if len(clusterName) > 100 || len(clusterName) == 0 {
		return false
	}

	if !unicode.IsLetter(rune(clusterName[0])) && !unicode.IsNumber(rune(clusterName[0])) {
		return false
	}

	return !strings.ContainsFunc(clusterName, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsNumber(r) && r != '-' && r != '_'
	})
}

func (cc *ConfigCache) getFalconImage(ctx context.Context, nodesensor *falconv1alpha1.FalconNodeSensor) (string, error) {
	if nodesensor.Spec.Node.Image != "" {
		return nodesensor.Spec.Node.Image, nil
	}

	nodeImage := os.Getenv("RELATED_IMAGE_NODE_SENSOR")
	if nodeImage != "" && cc.falconApiConfig == nil {
		return nodeImage, nil
	}

	if cc.falconApiConfig == nil {
		return "", ErrFalconAPINotConfigured
	}

	cloud, err := falcon_api.FalconCloud(ctx, cc.falconApiConfig)
	if err != nil {
		return "", err
	}

	var imageUri string
	isUsingCustomCrowdstrikeRepo := nodesensor.Spec.Internal.CrowdstrikeRegistryRepoOverride != nil

	if isUsingCustomCrowdstrikeRepo {
		imageUri = falcon_registry.CrowdstrikeRepoOverride(cloud, *nodesensor.Spec.Internal.CrowdstrikeRegistryRepoOverride)
	} else {
		imageUri = falcon_registry.ImageURINode(cloud)
		if nodesensor.Status.Sensor != nil {
			if falcon_registry.IsMinimumUnifiedSensorVersion(strings.Split(*nodesensor.Status.Sensor, "-")[0], falcon.NodeSensor) {
				imageUri = falcon_registry.UnifiedImageURINode(cloud)
			}
		}
	}

	if versionLock(nodesensor) {
		return fmt.Sprintf("%s:%s", imageUri, *nodesensor.Status.Sensor), nil
	}

	apiConfig := *cc.falconApiConfig
	apiConfig.Context = ctx
	imageRepo, err := sensor.NewImageRepository(ctx, &apiConfig)
	if err != nil {
		return "", err
	}

	if isUsingCustomCrowdstrikeRepo {
		imageRepo.SetOverrideImageUri(imageUri)
	}

	imageTag, err := imageRepo.GetPreferredImage(ctx, falcon.NodeSensor, nodesensor.Spec.Node.Version, nodesensor.Spec.Node.Advanced.UpdatePolicy)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s:%s", imageUri, imageTag), nil
}

func versionLock(nodesensor *falconv1alpha1.FalconNodeSensor) bool {
	if nodesensor.Status.Sensor == nil || nodesensor.Spec.Node.Advanced.HasUpdatePolicy() || nodesensor.Spec.Node.Advanced.IsAutoUpdating() {
		return false
	}

	return nodesensor.Spec.Node.Version == nil || strings.Contains(*nodesensor.Status.Sensor, *nodesensor.Spec.Node.Version)
}

func ConfigCacheTest(cid string, imageUri string, nodeTest *falconv1alpha1.FalconNodeSensor, apiConfig *falcon.ApiConfig) *ConfigCache {
	return &ConfigCache{
		cid:             cid,
		imageUri:        imageUri,
		nodesensor:      nodeTest,
		falconApiConfig: apiConfig,
	}
}
