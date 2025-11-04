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
			strings.Contains(tag, ".falcon-linux") &&
			(versionRequested == nil || strings.HasPrefix(tag, *versionRequested)))
	}

	var imageUri string
	if reg.falconOverrideRepo != "" {
		imageUri = reg.falconOverrideRepo
		filter = func(tag string) bool {
			return (tag[0] >= '0' && tag[0] <= '9' &&
				(versionRequested == nil || strings.HasPrefix(tag, *versionRequested)))
		}
	} else {
		if versionRequested == nil || (versionRequested != nil && IsMinimumUnifiedSensorVersion(strings.Split(*versionRequested, "-")[0])) {
			imageUri = UnifiedImageURINode(reg.falconCloud)
		} else {
			imageUri = ImageURINode(reg.falconCloud)
		}
	}

	return lastTag(ctx, systemContext, imageUri, filter)
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
