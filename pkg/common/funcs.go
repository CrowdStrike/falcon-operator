package common

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
)

func InitContainerArgs() []string {
	return []string{
		"-c",
		"touch " + FalconStoreFile,
	}
}

func GetFalconImage(nodesensor *falconv1alpha1.FalconNodeSensor) string {
	if nodesensor.Spec.Node.Image == "" {
		return FalconDefaultImage
	}
	return nodesensor.Spec.Node.Image
}

func FalconSensorConfig(falconsensor *falconv1alpha1.FalconSensor) (map[string]string, error) {
	m := make(map[string]string)
	var cmOptInt map[string]interface{}
	jsonCmOpt, err := json.Marshal(falconsensor)
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal(jsonCmOpt, &cmOptInt)
	if err != nil {
		return nil, err
	}

	// iterate through jsonCmOpt
	for field, val := range cmOptInt {
		if field != "" {
			// Make the keys match the env variable names for now
			key := "FALCONCTL_OPT_" + strings.ToUpper(field)

			switch v := val.(type) {
			case bool:
				m[key] = strconv.FormatBool(v)
			case string:
				m[key] = v
			default:
				return m, fmt.Errorf("unexpected type received for FALCONCTL_OPT, field '%s', type: %T, value: %v", field, val, val)
			}
		}
	}

	return m, nil
}
