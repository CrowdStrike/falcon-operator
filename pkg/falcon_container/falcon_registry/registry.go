package falcon_registry

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/docker/reference"
	"github.com/containers/image/v5/types"

	"github.com/crowdstrike/falcon-operator/pkg/falcon_api"
	"github.com/crowdstrike/falcon-operator/pkg/registry_auth"
	"github.com/crowdstrike/gofalcon/falcon"
)

type FalconRegistry struct {
	token       string
	falconCloud falcon.CloudType
	falconCID   string
}

func NewFalconRegistry(apiCfg *falcon.ApiConfig, CID string) (*FalconRegistry, error) {
	token, err := falcon_api.RegistryToken(apiCfg)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch registry token for CrowdStrike container registry: %v", err)
	}
	if token == "" {
		return nil, errors.New("Empty registry token received from CrowdStrike API")
	}

	return &FalconRegistry{
		falconCloud: apiCfg.Cloud,
		falconCID:   CID,
		token:       token,
	}, nil
}

func (reg *FalconRegistry) Pulltoken() ([]byte, error) {
	username, err := reg.username()
	if err != nil {
		return nil, err
	}
	dockerfile, err := registry_auth.Dockerfile("registry.crowdstrike.com", username, reg.token)
	if err != nil {
		return nil, err
	}
	return dockerfile, nil
}

func (reg *FalconRegistry) PullInfo(ctx context.Context, versionRequested *string) (falconTag string, falconImage types.ImageReference, systemContext *types.SystemContext, err error) {
	systemContext, err = reg.systemContext()
	if err != nil {
		return
	}
	falconTag, err = reg.LastContainerTag(ctx, versionRequested)
	if err != nil {
		return
	}
	falconImage, err = imageReference(reg.imageUri(), falconTag)
	if err != nil {
		return
	}
	return
}

func (reg *FalconRegistry) LastContainerTag(ctx context.Context, versionRequested *string) (string, error) {
	systemContext, err := reg.systemContext()
	if err != nil {
		return "", err
	}
	return lastTag(ctx, systemContext, reg.imageUri(), versionRequested)
}

func imageReference(imageUri, tag string) (types.ImageReference, error) {
	return docker.ParseReference(fmt.Sprintf("//%s:%s", imageUri, tag))
}

func lastTag(ctx context.Context, systemContext *types.SystemContext, imageUri string, versionRequested *string) (string, error) {
	ref, err := reference.ParseNormalizedNamed(imageUri)
	if err != nil {
		return "", err
	}
	imgRef, err := docker.NewReference(reference.TagNameOnly(ref))
	if err != nil {
		return "", err
	}

	tags, err := listDockerTags(ctx, systemContext, imgRef)
	if err != nil {
		return "", err
	}
	return guessLastTag(tags, versionRequested)
}

func guessLastTag(tags []string, versionRequested *string) (string, error) {
	versionTags := []string{}
	for _, tag := range tags {
		if tag[0] >= '0' && tag[0] <= '9' {
			versionTags = append(versionTags, tag)
		}
	}
	if len(versionTags) == 0 {
		return "", fmt.Errorf("Could not find suitable image tag in the CrowdStrike registry. Tags were: %+v", tags)
	}

	if versionRequested != nil {
		for i := range versionTags {
			tag := versionTags[len(versionTags)-i-1]
			if strings.HasPrefix(tag, *versionRequested) {
				return tag, nil
			}
		}
		return "", fmt.Errorf("Could not find suitable image tag in the CrowdStrike registry. Requested version was: %s while the available tags were: %+v", *versionRequested, tags)
	}
	return versionTags[len(versionTags)-1], nil
}

func listDockerTags(ctx context.Context, sys *types.SystemContext, imgRef types.ImageReference) ([]string, error) {
	tags, err := docker.GetRepositoryTags(ctx, sys, imgRef)
	if err != nil {
		return nil, fmt.Errorf("Error listing repository tags: %v", err)
	}
	return tags, nil
}

func (fr *FalconRegistry) systemContext() (*types.SystemContext, error) {
	username, err := fr.username()
	if err != nil {
		return nil, err
	}

	return &types.SystemContext{
		DockerAuthConfig: &types.DockerAuthConfig{
			Username: username,
			Password: fr.token,
		},
	}, nil
}

func (fr *FalconRegistry) username() (string, error) {
	s := strings.Split(fr.falconCID, "-")
	if len(s) != 2 {
		return "", fmt.Errorf("Cannot parse FalconCID. Expected exactly one '-' character in the '%s'", fr.falconCID)
	}
	lowerCID := strings.ToLower(s[0])
	return fmt.Sprintf("fc-%s", lowerCID), nil
}

func (fr *FalconRegistry) imageUri() string {
	return ImageURI(fr.falconCloud)
}

func ImageURI(falconCloud falcon.CloudType) string {
	return fmt.Sprintf("registry.crowdstrike.com/falcon-container/%s/release/falcon-sensor", falconCloud.String())
}
