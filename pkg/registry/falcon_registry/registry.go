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
	"github.com/crowdstrike/falcon-operator/pkg/registry/auth"
	"github.com/crowdstrike/gofalcon/falcon"
)

type FalconRegistry struct {
	token       string
	falconCloud falcon.CloudType
	falconCID   string
}

func NewFalconRegistry(apiCfg *falcon.ApiConfig) (*FalconRegistry, error) {
	client, err := falcon.NewClient(apiCfg)
	if err != nil {
		return nil, fmt.Errorf("Could not authenticate with CrowdStrike API: %v", err)
	}

	token, err := falcon_api.RegistryToken(apiCfg.Context, client)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch registry token for CrowdStrike container registry: %v", err)
	}
	if token == "" {
		return nil, errors.New("Empty registry token received from CrowdStrike API")
	}

	ccid, err := falcon_api.CCID(apiCfg.Context, client)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch CCID from CrowdStrike API: %v", err)
	}
	if ccid == "" {
		return nil, errors.New("Empty CCID received from CrowdStrike API")
	}

	return &FalconRegistry{
		falconCloud: apiCfg.Cloud,
		falconCID:   ccid,
		token:       token,
	}, nil
}

func (reg *FalconRegistry) Pulltoken() ([]byte, error) {
	username, err := reg.username()
	if err != nil {
		return nil, err
	}
	dockerfile, err := auth.Dockerfile("registry.crowdstrike.com", username, reg.token)
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

func imageReference(imageUri, tag string) (types.ImageReference, error) {
	return docker.ParseReference(fmt.Sprintf("//%s:%s", imageUri, tag))
}

func lastTag(ctx context.Context, systemContext *types.SystemContext, imageUri string, filter func(string) bool) (string, error) {
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
	return guessLastTag(tags, filter)
}

func guessLastTag(tags []string, filter func(string) bool) (string, error) {
	filteredTags := []string{}
	for _, tag := range tags {
		if filter(tag) {
			filteredTags = append(filteredTags, tag)
		}
	}
	if len(filteredTags) == 0 {
		return "", fmt.Errorf("Could not find suitable image tag in the CrowdStrike registry. Tags were: %+v", tags)
	}

	return filteredTags[len(filteredTags)-1], nil
}

func listDockerTags(ctx context.Context, sys *types.SystemContext, imgRef types.ImageReference) ([]string, error) {
	tags, err := docker.GetRepositoryTags(ctx, sys, imgRef)
	if err != nil {
		return nil, fmt.Errorf("Error listing repository (%s) tags: %v", imgRef.StringWithinTransport(), err)
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
