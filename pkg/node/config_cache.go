package node

import (
	"context"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/falcon_api"
	"github.com/crowdstrike/gofalcon/falcon"
	"github.com/go-logr/logr"
)

// ConfigCache holds config values for node sensor. Those values are either provided by user or fetched dynamically. That happens transparently to the caller.
type ConfigCache struct {
	cid string
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
