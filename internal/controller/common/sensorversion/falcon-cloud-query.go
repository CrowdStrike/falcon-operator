package sensorversion

import (
	"context"

	"github.com/crowdstrike/falcon-operator/pkg/registry/falcon_registry"
	"github.com/crowdstrike/gofalcon/falcon"
)

func NewFalconCloudQuery(sensorType falcon.SensorType, apiConfig *falcon.ApiConfig) SensorVersionQuery {
	return func(ctx context.Context) (string, error) {
		return getLatestSensorVersion(ctx, sensorType, apiConfig)
	}
}

func getLatestSensorVersion(ctx context.Context, sensorType falcon.SensorType, apiConfig *falcon.ApiConfig) (string, error) {
	if sensorType == falcon.NodeSensor || sensorType == falcon.RegionedNodeSensor {
		return getLatestSensorNodeVersion(ctx, apiConfig)
	}

	registry, err := falcon_registry.NewFalconRegistry(ctx, apiConfig)
	if err != nil {
		return "", err
	}

	return registry.LastContainerTag(ctx, sensorType, nil)
}

func getLatestSensorNodeVersion(ctx context.Context, apiConfig *falcon.ApiConfig) (string, error) {
	registry, err := falcon_registry.NewFalconRegistry(ctx, apiConfig)
	if err != nil {
		return "", err
	}

	return registry.LastNodeTag(ctx, nil)
}
