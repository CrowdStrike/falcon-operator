package node

import (
	"context"
	"fmt"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/falcon_api"
	"github.com/crowdstrike/gofalcon/falcon"
	"github.com/crowdstrike/falcon-operator/pkg/registry/falcon_registry"
	"github.com/go-logr/logr"
)

// ConfigCache holds config values for node sensor. Those values are either provided by user or fetched dynamically. That happens transparently to the caller.
type ConfigCache struct {
	cid string
	imageUri string
}

func (cc *ConfigCache) CID() string {
	return cc.cid
}



func NewConfigCache(ctx context.Context, logger logr.Logger, nodesensor *falconv1alpha1.FalconNodeSensor) (*ConfigCache, error) {
	var apiConfig *falcon.ApiConfig
	var err error
	cache := ConfigCache{}

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

func GetFalconImage(ctx context.Context, nodesensor *falconv1alpha1.FalconNodeSensor) (string, error) {
	if nodesensor.Spec.Node.ImageOverride != "" {
		return nodesensor.Spec.Node.ImageOverride, nil
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
	imageTag, err := registry.LastNodeTag(ctx, nil)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s:%s", imageUri, imageTag), nil
}
