package falcon_api

import (
	"fmt"
	"github.com/crowdstrike/gofalcon/falcon"
	"github.com/crowdstrike/gofalcon/falcon/client/falcon_container"
)

func RegistryToken(apiCfg *falcon.ApiConfig) (string, error) {
	client, err := falcon.NewClient(apiCfg)
	if err != nil {
		return "", err
	}

	res, err := client.FalconContainer.GetCredentials(&falcon_container.GetCredentialsParams{
		Context: apiCfg.Context,
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
