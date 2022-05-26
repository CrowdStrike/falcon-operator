package falcon_registry

import (
	"context"
	"fmt"
	"strings"

	"github.com/crowdstrike/gofalcon/falcon"
)

func (reg *FalconRegistry) LastContainerTag(ctx context.Context, versionRequested *string) (string, error) {
	systemContext, err := reg.systemContext()
	if err != nil {
		return "", err
	}

	return lastTag(ctx, systemContext, reg.imageUriContainer(), func(tag string) bool {
		return (tag[0] >= '0' && tag[0] <= '9' &&
			strings.Contains(tag, ".container.x86_64") &&
			(versionRequested == nil || strings.HasPrefix(tag, *versionRequested)))
	})
}

func ImageURIContainer(falconCloud falcon.CloudType) string {
	return fmt.Sprintf("%s/falcon-container/%s/release/falcon-sensor", registryFQDN(), falconCloud.String())
}

func (fr *FalconRegistry) imageUriContainer() string {
	return ImageURIContainer(fr.falconCloud)
}
