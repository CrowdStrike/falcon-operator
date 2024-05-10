package falcon_registry

import (
	"context"
	"strings"

	"github.com/crowdstrike/falcon-operator/pkg/common"
)

func (reg *FalconRegistry) LastContainerTag(ctx context.Context, sensorType common.SensorType, versionRequested *string) (string, error) {
	systemContext, err := reg.systemContext()
	if err != nil {
		return "", err
	}

	return lastTag(ctx, systemContext, reg.imageUriContainer(sensorType), func(tag string) bool {
		tagContains := ".container"

		return (tag[0] >= '0' && tag[0] <= '9' &&
			strings.Contains(tag, tagContains) &&
			(versionRequested == nil || strings.HasPrefix(tag, *versionRequested)))
	})
}

func (fr *FalconRegistry) imageUriContainer(sensorType common.SensorType) string {
	return SensorImageURI(fr.falconCloud, sensorType)
}
