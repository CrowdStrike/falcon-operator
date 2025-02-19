package falcon_registry

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"

	"github.com/containers/image/v5/docker"
	"github.com/containers/image/v5/docker/reference"
	"github.com/containers/image/v5/types"

	"github.com/crowdstrike/falcon-operator/pkg/falcon_api"
	"github.com/crowdstrike/falcon-operator/pkg/registry/auth"
	"github.com/crowdstrike/gofalcon/falcon"
	version "github.com/hashicorp/go-version"
)

type FalconRegistry struct {
	token       string
	falconCloud falcon.CloudType
	falconCID   string
}

func NewFalconRegistry(ctx context.Context, apiCfg *falcon.ApiConfig) (*FalconRegistry, error) {
	apiCfg.Context = ctx
	client, err := falcon.NewClient(apiCfg)
	if err != nil {
		return nil, fmt.Errorf("Could not authenticate with CrowdStrike API: %v", err)
	}

	token, err := falcon_api.RegistryToken(ctx, client)
	if err != nil {
		return nil, fmt.Errorf("Failed to fetch registry token for CrowdStrike container registry:, %v", err)
	}
	if token == "" {
		return nil, errors.New("Empty registry token received from CrowdStrike API")
	}

	ccid, err := falcon_api.CCID(ctx, client)
	if err != nil {
		return nil, err
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
	dockerfile, err := auth.Dockerfile(registryFQDN(reg.falconCloud), username, reg.token)
	if err != nil {
		return nil, err
	}
	return dockerfile, nil
}

func (reg *FalconRegistry) PullInfo(ctx context.Context, sensorType falcon.SensorType, versionRequested *string) (falconTag string, falconImage types.ImageReference, systemContext *types.SystemContext, err error) {
	systemContext, err = reg.systemContext()
	if err != nil {
		return
	}
	falconTag, err = reg.LastContainerTag(ctx, sensorType, versionRequested)
	if err != nil {
		return
	}
	falconImage, err = imageReference(reg.imageUriContainer(sensorType), falconTag)
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

	if strings.Contains(imageUri, "imageanalyzer") {
		sort.Slice(tags, func(i, j int) bool {
			v1, err1 := version.NewVersion(tags[i])
			v2, err2 := version.NewVersion(tags[j])
			if err1 != nil || err2 != nil {
				return tags[i] < tags[j]
			}
			return v1.LessThan(v2)
		})
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
		// Artifactory incorrectly adds the digest to the tag list, so we need to handle this case
		// by fetching the tags from the registry directly
		// This is a workaround, and should be removed once Artifactory is fixed
		if strings.Contains(err.Error(), "registry returned invalid tag") {
			var jsonRes map[string]interface{}
			reg, _, found := strings.Cut(imgRef.StringWithinTransport(), ":")
			if !found {
				return nil, fmt.Errorf("Error parsing repository (%s) from image reference: %v", imgRef.StringWithinTransport(), err)
			}

			if strings.Contains(reg, "crowdstrike.com/") {
				reg = strings.Replace(reg, "crowdstrike.com/", "crowdstrike.com/v2/", 1)
			} else {
				return nil, fmt.Errorf("Error parsing repository (%s) from image reference. Missing crowdstrike domain: %v", reg, err)
			}

			tr := &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion: tls.VersionTLS12,
				},
			}

			client := &http.Client{Transport: tr}
			req, err := http.NewRequest("GET", fmt.Sprintf("https:%s/tags/list", reg), nil)
			if err != nil {
				return nil, err
			}

			req.Header.Add("Accept", "application/json")
			req.Header.Add("Authorization", fmt.Sprintf("Bearer %s", sys.DockerAuthConfig.Password))
			resp, err := client.Do(req)
			if err != nil {
				return nil, fmt.Errorf("Unable to get http response to list container registry tags: %v", err)
			}
			defer resp.Body.Close()

			bodyText, err := io.ReadAll(resp.Body)
			if err != nil {
				return nil, fmt.Errorf("Unable to read response body to list registry tags: %v", err)
			}

			if err := json.Unmarshal(bodyText, &jsonRes); err != nil {
				return nil, fmt.Errorf("Unable to unmarshal JSON response list registry tags: %v", err)
			}

			jTags := jsonRes["tags"].([]interface{})
			if jTags == nil {
				return nil, fmt.Errorf("Unable to get tags from JSON response list registry tags: %v", err)
			}

			tagList := []string{}
			for _, tag := range jTags {
				if !strings.Contains(tag.(string), "sha256") {
					tagList = append(tagList, tag.(string))
				}
			}

			return tagList, nil
		}

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

func registryFQDN(cloud falcon.CloudType) string {
	switch cloud {
	case falcon.CloudUsGov1:
		return "registry.laggar.gcw.crowdstrike.com"
	default:
		return "registry.crowdstrike.com"
	}
}
