package falcon_registry

import (
	"context"
	"fmt"
	"strings"

	"github.com/crowdstrike/gofalcon/falcon"
	"go.podman.io/image/v5/types"
	"golang.org/x/mod/semver"
)

const (
	MinimumUnifiedNodeSensorVersion    = "7.31.0"
	MinimumUnifiedKacSensorVersion     = "7.33.0"
	MinimumUnifiedSidecarSensorVersion = "7.33.0"
	MinimumUnifiedImageSensorVersion   = "1.0.24"
)

func (reg *FalconRegistry) LastContainerTag(ctx context.Context, sensorType falcon.SensorType, versionRequested *string) (string, error) {
	var tag string

	systemContext, err := reg.systemContext()
	if err != nil {
		return "", err
	}

	regionedFilter := func(tag string) bool {
		tagContains := ".container"
		if sensorType == falcon.ImageSensor || sensorType == falcon.KacSensor {
			tagContains = ""
		}

		return (tag[0] >= '0' && tag[0] <= '9' &&
			strings.Contains(tag, tagContains) &&
			(versionRequested == nil || strings.HasPrefix(tag, *versionRequested)))
	}

	unifiedFilter := func(tag string) bool {
		return (tag[0] >= '0' && tag[0] <= '9' &&
			(versionRequested == nil || strings.HasPrefix(tag, *versionRequested)))
	}

	switch sensorType {
	case falcon.KacSensor:
		tag, err = reg.tryUnifiedThenRegioned(
			ctx, systemContext,
			falcon.KacSensor, falcon.RegionedKacSensor,
			unifiedFilter, regionedFilter,
		)
	case falcon.SidecarSensor:
		tag, err = reg.tryUnifiedThenRegioned(
			ctx, systemContext,
			falcon.SidecarSensor, falcon.RegionedSidecarSensor,
			unifiedFilter, regionedFilter,
		)
	case falcon.ImageSensor:
		tag, err = reg.tryUnifiedThenRegioned(
			ctx, systemContext,
			falcon.ImageSensor, falcon.RegionedImageSensor,
			unifiedFilter, regionedFilter,
		)
	default:
		tag, err = lastTag(ctx, systemContext, reg.imageUriContainer(sensorType), regionedFilter)
	}

	return tag, err
}

func (fr *FalconRegistry) imageUriContainer(sensorType falcon.SensorType) string {
	return falcon.FalconContainerSensorImageURI(fr.falconCloud, sensorType)
}

func IsMinimumUnifiedSensorVersion(version string, sensorType falcon.SensorType) bool {
	switch sensorType {
	case falcon.NodeSensor:
		return semver.Compare("v"+version, "v"+MinimumUnifiedNodeSensorVersion) >= 0
	case falcon.KacSensor:
		return semver.Compare("v"+version, "v"+MinimumUnifiedKacSensorVersion) >= 0
	case falcon.SidecarSensor:
		return semver.Compare("v"+version, "v"+MinimumUnifiedSidecarSensorVersion) >= 0
	case falcon.ImageSensor:
		return semver.Compare("v"+version, "v"+MinimumUnifiedImageSensorVersion) >= 0
	}

	return false
}

func (reg *FalconRegistry) tryUnifiedThenRegioned(
	ctx context.Context,
	systemContext *types.SystemContext,
	unifiedType, regionedType falcon.SensorType,
	unifiedFilter, regionedFilter func(string) bool,
) (string, error) {
	unifiedURI := falcon.FalconContainerSensorImageURI(reg.falconCloud, unifiedType)
	regionedURI := falcon.FalconContainerSensorImageURI(reg.falconCloud, regionedType)

	tag, err := lastTag(ctx, systemContext, unifiedURI, unifiedFilter)
	if err != nil {
		unifiedErr := fmt.Errorf("failed to fetch unified image sensor tag: %w", err)
		tag, err = lastTag(ctx, systemContext, regionedURI, regionedFilter)
		if err != nil {
			return "", fmt.Errorf("failed to fetch regioned image sensor tag: %w; previous error: %v", err, unifiedErr)
		}
	}
	return tag, nil
}
