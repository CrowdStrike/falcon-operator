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
		switch err.(type) {
		case *falcon_container.GetCredentialsForbidden:
			return "", fmt.Errorf("Insufficient CrowdStrike privileges, please grant [Falcon Images Download: Read] to CrowdStrike API Key. Error was: %s", err)
		}
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

func CCID(ctx context.Context, client *client.CrowdStrikeAPISpecification) (string, error) {
	response, err := client.SensorDownload.GetSensorInstallersCCIDByQuery(&sensor_download.GetSensorInstallersCCIDByQueryParams{
		Context: ctx,
	})
	if err != nil {
		switch err.(type) {
		case *sensor_download.GetSensorInstallersCCIDByQueryForbidden:
			return "", fmt.Errorf("Insufficient CrowdStrike privileges, please grant [Sensor Download: Read] to CrowdStrike API Key. Error was: %s", err)
		}
		return "", fmt.Errorf("Could not get CCID from CrowdStrike Falcon API: %v", err)
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
