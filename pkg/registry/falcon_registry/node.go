package falcon_registry

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/crowdstrike/gofalcon/falcon"
)

func (reg *FalconRegistry) LastNodeTag(ctx context.Context, versionRequested *string) (string, error) {
	systemContext, err := reg.systemContext()
	if err != nil {
		return "", err
	}

	return lastTag(ctx, systemContext, reg.imageUriNode(), func(tag string) bool {
		arch := "x86_64"
		if runtime.GOARCH == "arm64" {
			arch = "aarch64"
		}
		return (tag[0] >= '0' && tag[0] <= '9' &&
			strings.Contains(tag, fmt.Sprintf(".falcon-linux.%s", arch)) &&
			(versionRequested == nil || strings.HasPrefix(tag, *versionRequested)))
	})
}

func ImageURINode(falconCloud falcon.CloudType) string {
	return fmt.Sprintf("%s/falcon-sensor/%s/release/falcon-sensor", registryFQDN(falconCloud), registryCloud(falconCloud))
}

func (fr *FalconRegistry) imageUriNode() string {
	return ImageURINode(fr.falconCloud)
}
