package falcon_registry

import (
	"context"
	"fmt"
	"strings"

	"github.com/crowdstrike/gofalcon/falcon"
)

const (
	// ClusterGuardRepo is the CrowdStrike registry repository for the Cluster Guard image.
	// Cluster Guard is temporarily published to the unified falcon-sensor repository.
	ClusterGuardRepo = "falcon-sensor"

	defaultBuildStage = "release"
)

// LastClusterGuardTag returns the newest Cluster Guard image tag in the given build stage matching versionRequested.
func (reg *FalconRegistry) LastClusterGuardTag(ctx context.Context, buildStage string, versionRequested *string) (string, error) {
	systemContext, err := reg.systemContext()
	if err != nil {
		return "", err
	}

	filter := func(tag string) bool {
		return (tag[0] >= '0' && tag[0] <= '9' &&
			(versionRequested == nil || strings.HasPrefix(tag, *versionRequested)))
	}

	return lastTag(ctx, systemContext, ClusterGuardImageURI(reg.falconCloud, buildStage), filter)
}

// ClusterGuardImageURI returns the Cluster Guard image repository URI for the given cloud and build stage.
// An empty build stage defaults to release.
func ClusterGuardImageURI(falconCloud falcon.CloudType, buildStage string) string {
	if buildStage == "" {
		buildStage = defaultBuildStage
	}
	return fmt.Sprintf("%s/%s/%s/%s", registryFQDN(falconCloud), ClusterGuardRepo, buildStage, ClusterGuardRepo)
}
