package sensor

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strings"

	"github.com/crowdstrike/falcon-operator/pkg/registry/falcon_registry"
	"github.com/crowdstrike/gofalcon/falcon"
	"github.com/crowdstrike/gofalcon/falcon/client/sensor_update_policies"
	"github.com/crowdstrike/gofalcon/falcon/models"
	"github.com/go-logr/logr"
	"github.com/go-openapi/swag"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const amd64 = "amd64"
const arm64 = "arm64"
const arm64Platform = "LinuxArm64"

var (
	errInvalidSensorVersion  = errors.New("invalid sensor version")
	errSensorVersionNotFound = errors.New("sensor version not found")
)

type ImageRepository struct {
	api                   sensorUpdatePoliciesAPI
	getSystemArchitecture func() string
	tags                  tagRegistry
}

func NewImageRepository(ctx context.Context, apiConfig *falcon.ApiConfig) (ImageRepository, error) {
	apiClient, err := falcon.NewClient(apiConfig)
	if err != nil {
		return ImageRepository{}, err
	}

	registry, err := falcon_registry.NewFalconRegistry(ctx, apiConfig)
	if err != nil {
		return ImageRepository{}, err
	}

	return ImageRepository{
		api:                   apiClient.SensorUpdatePolicies,
		getSystemArchitecture: func() string { return runtime.GOARCH },
		tags:                  registry,
	}, nil
}

func (images ImageRepository) GetPreferredImage(ctx context.Context, sensorType falcon.SensorType, versionSpec *string, updatePolicySpec *string) (string, error) {
	logger := log.FromContext(ctx).
		WithValues("architecture", images.getSystemArchitecture()).
		WithValues("sensorType", sensorType)

	version, err := images.getPreferredSensorVersion(versionSpec, updatePolicySpec, logger)
	if err != nil {
		return "", err
	}

	tag, err := images.getImageTagForSensorVersion(ctx, sensorType, version)
	if err != nil {
		return "", err
	}

	logger.Info("selected sensor image", "tag", tag)
	return tag, nil
}

func (images ImageRepository) findPolicy(policyName string) (string, error) {
	filter := falconFilter{}.
		addClause("platform_name", "Linux").
		addClause("name.raw", policyName).
		encode()

	params := sensor_update_policies.NewQuerySensorUpdatePoliciesParams().WithFilter(&filter)
	response, err := images.api.QuerySensorUpdatePolicies(params)
	if err != nil {
		return "", err
	}

	ids := getNonZeroValuesInSlice(response.Payload.Resources)
	if len(ids) == 0 {
		return "", fmt.Errorf("update-policy %s not found", policyName)
	}

	return ids[0], nil
}

func (images ImageRepository) findSensorVersionByUpdatePolicy(updatePolicy string) (string, error) {
	policyID, err := images.findPolicy(updatePolicy)
	if err != nil {
		return "", err
	}

	version, err := images.getSensorVersionForPolicy(policyID)
	if err == errInvalidSensorVersion {
		return "", fmt.Errorf("update-policy with ID %s has an invalid sensor version", policyID)
	} else if err == errSensorVersionNotFound {
		return "", fmt.Errorf("update-policy with ID %s contains no version for system architecture %s", policyID, images.getSystemArchitecture())
	} else if err != nil {
		return "", err
	}

	return version, nil
}

func (images ImageRepository) getImageTagForSensorVersion(ctx context.Context, sensorType falcon.SensorType, version *string) (string, error) {
	if sensorType == falcon.NodeSensor {
		return images.tags.LastNodeTag(ctx, version)
	}

	return images.tags.LastContainerTag(ctx, sensorType, version)
}

func (images ImageRepository) getPreferredSensorVersion(versionSpec *string, updatePolicySpec *string, logger logr.Logger) (*string, error) {
	if versionSpec != nil && *versionSpec != "" {
		logger.Info("requested specific sensor version", "version", *versionSpec)
		return versionSpec, nil
	}

	if updatePolicySpec != nil && *updatePolicySpec != "" {
		logger.Info("requested sensor update policy", "policyName", *updatePolicySpec)
		version, err := images.findSensorVersionByUpdatePolicy(*updatePolicySpec)
		if err != nil {
			return nil, err
		}

		logger.Info("version selected by sensor update policy", "policyName", *updatePolicySpec, "version", version)
		return &version, nil
	}

	logger.Info("requested latest sensor version")
	return nil, nil
}

func (images ImageRepository) getSensorVersionForCurrentRuntimeArchitecture(policy *models.SensorUpdatePolicyV2) (string, error) {
	switch images.getSystemArchitecture() {
	case amd64:
		return trimVersion(policy.Settings.SensorVersion)
	case arm64:
		return getARM64Variant(policy)
	}

	return "", errSensorVersionNotFound
}

func (images ImageRepository) getSensorVersionForPolicy(policyID string) (string, error) {
	params := sensor_update_policies.NewGetSensorUpdatePoliciesV2Params().WithIds([]string{policyID})
	response, err := images.api.GetSensorUpdatePoliciesV2(params)
	if err != nil {
		return "", err
	}

	policies := getNonZeroValuesInSlice(response.Payload.Resources)
	if len(policies) == 0 {
		return "", fmt.Errorf("update-policy with ID %s not found", policyID)
	}

	policy := policies[0]
	if !*policy.Enabled {
		return "", fmt.Errorf("update-policy with ID %s is disabled", policyID)
	}

	return images.getSensorVersionForCurrentRuntimeArchitecture(policy)
}

func getARM64Variant(policy *models.SensorUpdatePolicyV2) (string, error) {
	for _, variant := range policy.Settings.Variants {
		if *variant.Platform == arm64Platform {
			return trimVersion(variant.SensorVersion)
		}
	}

	return "", errSensorVersionNotFound
}

func getNonZeroValuesInSlice[T any](input []T) []T {
	output := make([]T, 0)

	for _, value := range input {
		if !swag.IsZero(value) {
			output = append(output, value)
		}
	}

	return output
}

func trimVersion(version *string) (string, error) {
	if version == nil {
		return "", errSensorVersionNotFound
	}

	trimmed := strings.TrimSpace(*version)
	if trimmed == "" {
		return "", errSensorVersionNotFound
	}

	parts := strings.Split(trimmed, ".")
	if len(parts) != 3 {
		return "", errInvalidSensorVersion
	}

	return strings.Join(parts[0:2], "."), nil
}

type falconFilter struct {
	clauses []string
}

func (filter falconFilter) addClause(name string, value string) falconFilter {
	filter.clauses = append(filter.clauses, fmt.Sprintf(`%s:"%s"`, name, value))
	return filter
}

func (filter falconFilter) encode() string {
	return strings.Join(filter.clauses, "+")
}

type sensorUpdatePoliciesAPI interface {
	GetSensorUpdatePoliciesV2(params *sensor_update_policies.GetSensorUpdatePoliciesV2Params, opts ...sensor_update_policies.ClientOption) (*sensor_update_policies.GetSensorUpdatePoliciesV2OK, error)
	QuerySensorUpdatePolicies(params *sensor_update_policies.QuerySensorUpdatePoliciesParams, opts ...sensor_update_policies.ClientOption) (*sensor_update_policies.QuerySensorUpdatePoliciesOK, error)
}

type tagRegistry interface {
	LastContainerTag(ctx context.Context, sensorType falcon.SensorType, versionRequested *string) (string, error)
	LastNodeTag(ctx context.Context, versionRequested *string) (string, error)
}
