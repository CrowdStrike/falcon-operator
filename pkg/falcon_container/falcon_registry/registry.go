package falcon_registry

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/go-logr/logr"

	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/docker/reference"
	"github.com/containers/image/v5/types"

	"github.com/crowdstrike/gofalcon/falcon"
	"github.com/crowdstrike/gofalcon/falcon/client/falcon_container"
)

type FalconRegistry struct {
	token       string
	falconCloud falcon.CloudType
	falconCID   string
}

func NewFalconRegistry(apiCfg *falcon.ApiConfig, CID string, logger logr.Logger) (*FalconRegistry, error) {
	token, err := registryToken(apiCfg, logger)
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

func (reg *FalconRegistry) PullInfo(ctx context.Context) (falconTag string, falconImage types.ImageReference, systemContext *types.SystemContext, err error) {
	systemContext, err = reg.systemContext()
	if err != nil {
		return
	}
	imageUri, err := reg.imageUri()
	if err != nil {
		return
	}
	falconTag, err = lastTag(ctx, systemContext, imageUri)
	if err != nil {
		return
	}

	falconImage, err = imageReference(imageUri, falconTag)
	if err != nil {
		return
	}
	return
}

func imageReference(imageUri, tag string) (types.ImageReference, error) {
	return docker.ParseReference(fmt.Sprintf("//%s:%s", imageUri, tag))
}

func lastTag(ctx context.Context, systemContext *types.SystemContext, imageUri string) (string, error) {
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
	return guessLastTag(tags)
}

func guessLastTag(tags []string) (string, error) {
	versionTags := []string{}
	for _, tag := range tags {
		if tag[0] >= '0' && tag[0] <= '9' {
			versionTags = append(versionTags, tag)
		}
	}
	if len(versionTags) == 0 {
		return "", fmt.Errorf("Could not find suitable image tag in the CrowdStrike registry. Tags were: %+v", tags)
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

func registryToken(apiCfg *falcon.ApiConfig, logger logr.Logger) (string, error) {
	client, err := falcon.NewClient(apiCfg)
	if err != nil {
		return "", err
	}

	res, err := client.FalconContainer.GetCredentials(&falcon_container.GetCredentialsParams{
		Context: context.Background(),
	})
	if err != nil {
		return "", err
	}
	payload := res.GetPayload()
	if err = falcon.AssertNoError(payload.Errors); err != nil {
		return "", err
	}
	resources := payload.Resources
	resourcesList := resources.([]interface{})
	if len(resourcesList) != 1 {
		return "", fmt.Errorf("Expected to receive exactly one token, but got %d\n", len(resourcesList))
	}
	resourceMap := resourcesList[0].(map[string]interface{})
	value, ok := resourceMap["token"]
	if !ok {
		return "", fmt.Errorf("Expected to receive map containing 'token' key, but got %s\n", resourceMap)
	}
	valueString := value.(string)
	return valueString, nil
}

func (fr *FalconRegistry) imageUri() (string, error) {
	cloud, err := fr.falconCloudLower()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("registry.crowdstrike.com/falcon-container/%s/release/falcon-sensor", cloud), nil
}

func (fr *FalconRegistry) falconCloudLower() (string, error) {
	switch fr.falconCloud {
	case falcon.CloudUs1:
		return "us-1", nil
	case falcon.CloudUs2:
		return "us-2", nil
	case falcon.CloudEu1:
		return "eu-1", nil
	}
	return "", fmt.Errorf("Unrecognized Falcon Cloud Region: %v", fr.falconCloud)
}
