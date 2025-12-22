package falcon_registry

import (
	"context"
	"strings"

	"github.com/crowdstrike/gofalcon/falcon"
	"golang.org/x/mod/semver"
)

const (
	MinimumUnifiedNodeSensorVersion    = "7.31.0"
	MinimumUnifiedKacSensorVersion     = "7.33.0"
	MinimumUnifiedSidecarSensorVersion = "7.33.0"
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
		tag, err = lastTag(ctx, systemContext, falcon.FalconContainerSensorImageURI(reg.falconCloud, falcon.KacSensor), unifiedFilter)
		if err != nil {
			tag, err = lastTag(ctx, systemContext, falcon.FalconContainerSensorImageURI(reg.falconCloud, falcon.RegionedKacSensor), regionedFilter)
		}
	case falcon.SidecarSensor:
		tag, err = lastTag(ctx, systemContext, falcon.FalconContainerSensorImageURI(reg.falconCloud, falcon.SidecarSensor), unifiedFilter)
		if err != nil {
			tag, err = lastTag(ctx, systemContext, falcon.FalconContainerSensorImageURI(reg.falconCloud, falcon.RegionedSidecarSensor), regionedFilter)
		}
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
	}

	return false
}
