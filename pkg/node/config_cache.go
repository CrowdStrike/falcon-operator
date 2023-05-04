package node

import (
	"context"
	"fmt"
	"os"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/crowdstrike/falcon-operator/pkg/falcon_api"
	"github.com/crowdstrike/falcon-operator/pkg/registry/falcon_registry"
	"github.com/crowdstrike/falcon-operator/pkg/registry/pulltoken"
	"github.com/crowdstrike/gofalcon/falcon"
	"github.com/go-logr/logr"
)

// ConfigCache holds config values for node sensor. Those values are either provided by user or fetched dynamically. That happens transparently to the caller.
type ConfigCache struct {
	cid        string
	imageUri   string
	nodesensor *falconv1alpha1.FalconNodeSensor
}

func (cc *ConfigCache) CID() string {
	return cc.cid
}

func (cc *ConfigCache) UsingCrowdStrikeRegistry() bool {
	if cc.nodesensor.Spec.Node.Image == "" && cc.nodesensor.Spec.FalconAPI == nil {
		return os.Getenv("RELATED_IMAGE_NODE_SENSOR") == ""
	}
	return cc.nodesensor.Spec.Node.Image == ""
}

func (cc *ConfigCache) GetImageURI(ctx context.Context, logger logr.Logger) (string, error) {
	var err error
	if cc.imageUri == "" {
		cc.imageUri, err = getFalconImage(ctx, cc.nodesensor)
		if err == nil {
			logger.Info("Identified Falcon Node Image", "reference", cc.imageUri)
		}
	}
	return cc.imageUri, err
}

func (cc *ConfigCache) GetPullToken(ctx context.Context) ([]byte, error) {
	if cc.nodesensor.Spec.FalconAPI == nil {
		return nil, fmt.Errorf("Missing falcon_api configuration")
	}
	return pulltoken.CrowdStrike(ctx, cc.nodesensor.Spec.FalconAPI.ApiConfig())
}

func (cc *ConfigCache) SensorEnvVars() map[string]string {
	sensorConfig := common.MakeSensorEnvMap(cc.nodesensor.Spec.Falcon)
	if cc.cid != "" {
		sensorConfig["FALCONCTL_OPT_CID"] = cc.cid
	}
	if cc.nodesensor.Spec.Node.Backend != "" {
		sensorConfig["FALCONCTL_OPT_BACKEND"] = cc.nodesensor.Spec.Node.Backend
	}
	return sensorConfig
}

func NewConfigCache(ctx context.Context, logger logr.Logger, nodesensor *falconv1alpha1.FalconNodeSensor) (*ConfigCache, error) {
	var apiConfig *falcon.ApiConfig
	var err error
	cache := ConfigCache{
		nodesensor: nodesensor,
	}

	if nodesensor.Spec.FalconAPI != nil {
		apiConfig = nodesensor.Spec.FalconAPI.ApiConfig()
		if nodesensor.Spec.FalconAPI.CID != nil {
			cache.cid = *nodesensor.Spec.FalconAPI.CID
		}
	}

	if cache.cid == "" {
		cache.cid, err = falcon_api.FalconCID(ctx, nodesensor.Spec.Falcon.CID, apiConfig)
		if err != nil {
			return nil, err
		}
	}

	return &cache, nil
}

func getFalconImage(ctx context.Context, nodesensor *falconv1alpha1.FalconNodeSensor) (string, error) {
	if nodesensor.Spec.Node.Image != "" {
		return nodesensor.Spec.Node.Image, nil
	}

	nodeImage := os.Getenv("RELATED_IMAGE_NODE_SENSOR")
	if nodeImage != "" && nodesensor.Spec.FalconAPI == nil {
		return nodeImage, nil
	}

	if nodesensor.Spec.FalconAPI == nil {
		return "", fmt.Errorf("Missing falcon_api configuration")
	}

	cloud, err := nodesensor.Spec.FalconAPI.FalconCloud(ctx)
	if err != nil {
		return "", err
	}
	imageUri := falcon_registry.ImageURINode(cloud)

	registry, err := falcon_registry.NewFalconRegistry(ctx, nodesensor.Spec.FalconAPI.ApiConfig())
	if err != nil {
		return "", err
	}
	imageTag, err := registry.LastNodeTag(ctx, nodesensor.Spec.Node.Version)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s:%s", imageUri, imageTag), nil
}

func ConfigCacheTest(cid string, imageUri string, nodeTest *falconv1alpha1.FalconNodeSensor) *ConfigCache {
	return &ConfigCache{
		cid:        cid,
		imageUri:   imageUri,
		nodesensor: nodeTest,
	}
}
