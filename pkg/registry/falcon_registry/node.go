package falcon_registry

import (
	"context"
	"strings"

	"github.com/crowdstrike/gofalcon/falcon"
)

func (reg *FalconRegistry) LastNodeTag(ctx context.Context, versionRequested *string) (string, error) {
	systemContext, err := reg.systemContext()
	if err != nil {
		return "", err
	}

	return lastTag(ctx, systemContext, reg.imageUriNode(), func(tag string) bool {
		return (tag[0] >= '0' && tag[0] <= '9' &&
			strings.Contains(tag, ".falcon-linux.x86_64") &&
			(versionRequested == nil || strings.HasPrefix(tag, *versionRequested)))
	})
}

func ImageURINode(falconCloud falcon.CloudType) string {
	return falcon.FalconContainerSensorImageURI(falconCloud, falcon.NodeSensor)
}

func (fr *FalconRegistry) imageUriNode() string {
	return ImageURINode(fr.falconCloud)
}
