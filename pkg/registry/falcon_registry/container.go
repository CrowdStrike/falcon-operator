package falcon_registry

import (
	"context"
	"strings"

	"github.com/crowdstrike/gofalcon/falcon"
)

func (reg *FalconRegistry) LastContainerTag(ctx context.Context, sensorType falcon.SensorType, versionRequested *string) (string, error) {
	systemContext, err := reg.systemContext()
	if err != nil {
		return "", err
	}

	return lastTag(ctx, systemContext, reg.imageUriContainer(sensorType), func(tag string) bool {
		tagContains := ".container"
		if sensorType == falcon.ImageSensor || sensorType == falcon.KacSensor {
			tagContains = ""
		}

		return (tag[0] >= '0' && tag[0] <= '9' &&
			strings.Contains(tag, tagContains) &&
			(versionRequested == nil || strings.HasPrefix(tag, *versionRequested)))
	})
}

func (fr *FalconRegistry) imageUriContainer(sensorType falcon.SensorType) string {
	return falcon.FalconContainerSensorImageURI(fr.falconCloud, sensorType)
}
