package falcon_registry

import (
	"context"
	"strings"

	"github.com/crowdstrike/gofalcon/falcon"
)

func (reg *FalconRegistry) LastImageAnalyzerTag(ctx context.Context, sensorType falcon.SensorType, versionRequested *string) (string, error) {
	systemContext, err := reg.systemContext()
	if err != nil {
		return "", err
	}

	return lastTag(ctx, systemContext, reg.imageUriImageAnalyzer(sensorType), func(tag string) bool {
		return (tag[0] >= '0' && tag[0] <= '9' &&
			strings.Contains(tag, "1.0.8") &&
			(versionRequested == nil || strings.HasPrefix(tag, *versionRequested)))
	})
}

func (fr *FalconRegistry) imageUriImageAnalyzer(sensorType falcon.SensorType) string {
	return falcon.FalconContainerSensorImageURI(fr.falconCloud, sensorType)
}
