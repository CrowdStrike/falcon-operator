package node

import (
	"context"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/falcon_api"
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
	cid, err := falcon_api.FalconCID(ctx, nodesensor.Spec.Falcon.CID, nodesensor.Spec.FalconAPI.ApiConfig())
	if err != nil {
		return nil, err
	}

	return &ConfigCache{
		cid: cid,
	}, nil
}
