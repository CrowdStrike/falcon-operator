package falcon_registry

import (
	"context"
	"fmt"
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
			strings.Contains(tag, ".falcon-linux") &&
			(versionRequested == nil || strings.HasPrefix(tag, *versionRequested)))
	})
}

func ImageURINode(falconCloud falcon.CloudType) string {
	return fmt.Sprintf("%s/falcon-sensor/%s/release/falcon-sensor", registryFQDN(falconCloud), registryCloud(falconCloud))
}

func UnifiedImageURINode(falconCloud falcon.CloudType) string {
	return falcon.FalconContainerSensorImageURI(falconCloud, falcon.NodeSensor)
}

func (fr *FalconRegistry) imageUriNode() string {
	return ImageURINode(fr.falconCloud)
}
