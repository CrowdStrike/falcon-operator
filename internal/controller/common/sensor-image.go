package common

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/crowdstrike/falcon-operator/pkg/registry/falcon_registry"
	"github.com/crowdstrike/gofalcon/falcon"
	"github.com/crowdstrike/gofalcon/falcon/client"
	"github.com/crowdstrike/gofalcon/falcon/client/sensor_update_policies"
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const DefaultSensorUpdateFrequency = time.Hour * 24

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

func GetPreferredSensorImage(ctx context.Context, sensorType falcon.SensorType, versionSpec *string, updatePolicySpec *string, apiConfig *falcon.ApiConfig) (string, error) {
	logger := log.FromContext(ctx).
		WithValues("sensorType", sensorType)

	if apiConfig == nil {
		return "", errors.New("FalconAPI not set -- cannot lookup sensor image")
	}

	version, err := getPreferredSensorVersion(versionSpec, updatePolicySpec, apiConfig, logger)
	if err != nil {
		return "", err
	}

	registry, err := falcon_registry.NewFalconRegistry(ctx, apiConfig)
	if err != nil {
		return "", err
	}

	tag, err := registry.LastContainerTag(ctx, sensorType, version)
	if err != nil {
		return "", err
	}

	logger.Info("selected sensor image", "tag", tag)
	return tag, nil
}

func GetPreferredSensorNodeImage(ctx context.Context, versionSpec *string, updatePolicySpec *string, apiConfig *falcon.ApiConfig) (string, error) {
	logger := log.FromContext(ctx).
		WithValues("sensorType", "node")

	if apiConfig == nil {
		return "", errors.New("FalconAPI not set -- cannot lookup sensor image")
	}

	version, err := getPreferredSensorVersion(versionSpec, updatePolicySpec, apiConfig, logger)
	if err != nil {
		return "", err
	}

	registry, err := falcon_registry.NewFalconRegistry(ctx, apiConfig)
	if err != nil {
		return "", err
	}

	tag, err := registry.LastNodeTag(ctx, version)
	if err != nil {
		return "", err
	}

	logger.Info("selected sensor image", "tag", tag)
	return tag, nil
}

func GetSensorUpdateFrequency(configProperty *int32) time.Duration {
	if configProperty == nil || *configProperty <= 0 {
		return DefaultSensorUpdateFrequency
	}

	return time.Second * time.Duration(*configProperty)
}

func findPolicy(policyName string, apiClient *client.CrowdStrikeAPISpecification) (string, error) {
	filter := falconFilter{}.
		addClause("platform_name", "Linux").
		addClause("name.raw", policyName).
		encode()

	params := sensor_update_policies.NewQuerySensorUpdatePoliciesParams().WithFilter(&filter)
	response, err := apiClient.SensorUpdatePolicies.QuerySensorUpdatePolicies(params)
	if err != nil {
		return "", err
	} else if len(response.Payload.Resources) < 1 {
		return "", fmt.Errorf("update-policy %s not found", policyName)
	}

	policyID := response.Payload.Resources[0]
	return policyID, nil
}

func findSensorVersionByUpdatePolicy(updatePolicy string, apiConfig *falcon.ApiConfig) (string, error) {
	apiClient, err := falcon.NewClient(apiConfig)
	if err != nil {
		return "", err
	}

	policyID, err := findPolicy(updatePolicy, apiClient)
	if err != nil {
		return "", err
	}

	version, err := getSensorVersionForPolicy(policyID, apiClient)
	if err != nil {
		return "", err
	}

	return version, nil
}

func getPreferredSensorVersion(versionSpec *string, updatePolicySpec *string, apiConfig *falcon.ApiConfig, logger logr.Logger) (*string, error) {
	if versionSpec != nil && *versionSpec != "" {
		logger.Info("requested specific sensor version", "version", *versionSpec)
		return versionSpec, nil
	}

	if updatePolicySpec != nil && *updatePolicySpec != "" {
		logger.Info("requested sensor update policy", "policyName", *updatePolicySpec)
		version, err := findSensorVersionByUpdatePolicy(*updatePolicySpec, apiConfig)
		if err != nil {
			return nil, err
		}

		logger.Info("version selected by sensor update policy", "policyName", *updatePolicySpec, "version", version)
		return &version, nil
	}

	logger.Info("requested latest sensor version")
	return nil, nil
}

func getSensorVersionForPolicy(policyID string, apiClient *client.CrowdStrikeAPISpecification) (string, error) {
	params := sensor_update_policies.NewGetSensorUpdatePoliciesV2Params().WithIds([]string{policyID})
	response, err := apiClient.SensorUpdatePolicies.GetSensorUpdatePoliciesV2(params)
	if err != nil {
		return "", err
	} else if len(response.GetPayload().Resources) < 1 {
		return "", fmt.Errorf("update-policy ID %s not found", policyID)
	}

	sensorVersion := response.GetPayload().Resources[0].Settings.SensorVersion
	if sensorVersion == nil {
		return "", fmt.Errorf("update-policy ID %s has no sensor version", policyID)
	}

	parts := strings.Split(*sensorVersion, ".")
	if len(parts) != 3 {
		return "", fmt.Errorf("update-policy ID %s has an invalid sensor version", policyID)
	}

	return strings.Join(parts[0:2], "."), nil
}
