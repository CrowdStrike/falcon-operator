package falcon_registry

import (
	"context"
	"fmt"
	"strings"

	"github.com/crowdstrike/gofalcon/falcon"
	"golang.org/x/mod/semver"
)

const (
	MinimumUnifiedSensorVersion = "7.31.0"
)

func (reg *FalconRegistry) LastNodeTag(ctx context.Context, versionRequested *string) (string, error) {
	systemContext, err := reg.systemContext()
	if err != nil {
		return "", err
	}

	filter := func(tag string) bool {
		return (tag[0] >= '0' && tag[0] <= '9' &&
			(versionRequested == nil || strings.HasPrefix(tag, *versionRequested)))
	}

	if reg.falconOverrideRepo != "" {
		imageUri := reg.falconOverrideRepo
		return lastTag(ctx, systemContext, imageUri, filter)
	}

	tag, err := lastTag(ctx, systemContext, UnifiedImageURINode(reg.falconCloud), filter)
	if err != nil {
		return lastTag(ctx, systemContext, ImageURINode(reg.falconCloud), filter)
	}

	return tag, err
}

func (reg *FalconRegistry) SetCrowdstrikeRepoOverride(repo string) {
	reg.falconOverrideRepo = repo
}

func ImageURINode(falconCloud falcon.CloudType) string {
	return falcon.FalconContainerSensorImageURI(falconCloud, falcon.RegionedNodeSensor)
}

func UnifiedImageURINode(falconCloud falcon.CloudType) string {
	return falcon.FalconContainerSensorImageURI(falconCloud, falcon.NodeSensor)
}

func CrowdstrikeRepoOverride(falconCloud falcon.CloudType, repoOverride string) string {
	return fmt.Sprintf("%s/%s", registryFQDN(falconCloud), repoOverride)
}

func IsMinimumUnifiedSensorVersion(version string) bool {
	return semver.Compare("v"+version, "v"+MinimumUnifiedSensorVersion) >= 0
}
