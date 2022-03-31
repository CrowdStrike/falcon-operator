package falcon_api

import (
	"context"
	"fmt"

	"github.com/crowdstrike/gofalcon/falcon"
	"github.com/crowdstrike/gofalcon/falcon/client"
	"github.com/crowdstrike/gofalcon/falcon/client/falcon_container"
	"github.com/crowdstrike/gofalcon/falcon/client/sensor_download"
)

func RegistryToken(ctx context.Context, client *client.CrowdStrikeAPISpecification) (string, error) {
	res, err := client.FalconContainer.GetCredentials(&falcon_container.GetCredentialsParams{
		Context: ctx,
	})
	if err != nil {
		return "", errorHint(err, "")
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

func CCID(ctx context.Context, client *client.CrowdStrikeAPISpecification) (string, error) {
	response, err := client.SensorDownload.GetSensorInstallersCCIDByQuery(&sensor_download.GetSensorInstallersCCIDByQueryParams{
		Context: ctx,
	})
	if err != nil {
		return "", errorHint(err, "Could not get CCID from CrowdStrike Falcon API")
	}
	payload := response.GetPayload()
	if err = falcon.AssertNoError(payload.Errors); err != nil {
		return "", fmt.Errorf("Error reported when getting CCID from CrowdStrike Falcon API: %v", err)
	}
	if len(payload.Resources) != 1 {
		return "", fmt.Errorf("Failed to get CCID: Unexpected API response: %v", payload.Resources)
	}
	return payload.Resources[0], nil

}

func FalconCID(ctx context.Context, cid *string, fa *falcon.ApiConfig) (string, error) {
	fa.Context = ctx
	if cid != nil {
		return *cid, nil
	}

	client, err := falcon.NewClient(fa)
	if err != nil {
		return "", err
	}
	return CCID(ctx, client)
}

// FalconCloud returns user's Falcon Cloud based on supplied ApiConfig. This method will run cloud autodiscovery if 'autodiscover' is set in the ApiConfig
func FalconCloud(ctx context.Context, fa *falcon.ApiConfig) (falcon.CloudType, error) {
	err := fa.Cloud.Autodiscover(ctx, fa.ClientId, fa.ClientSecret)
	if err != nil {
		return fa.Cloud, errorHint(err, "Could not autodiscover Falcon Cloud Region. Please provide your cloud_region in FalconContainer Spec")
	}
	return fa.Cloud, nil
}
