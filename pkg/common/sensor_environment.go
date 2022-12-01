package common

import (
	"strconv"
	"strings"

	"github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
)

func MakeSensorEnvMap(falconSensor v1alpha1.FalconSensor) map[string]string {
	sensorConfig := make(map[string]string)

	if falconSensor.APD != nil {
		sensorConfig["FALCONCTL_OPT_APD"] = strconv.FormatBool(*falconSensor.APD)
	}
	if falconSensor.APH != "" {
		sensorConfig["FALCONCTL_OPT_APH"] = falconSensor.APH
	}
	if falconSensor.APP != nil {
		sensorConfig["FALCONCTL_OPT_APP"] = strconv.Itoa(*falconSensor.APP)
	}
	if falconSensor.Billing != "" {
		sensorConfig["FALCONCTL_OPT_BILLING"] = falconSensor.Billing
	}
	if falconSensor.PToken != "" {
		sensorConfig["FALCONCTL_OPT_PROVISIONING_TOKEN"] = falconSensor.PToken
	}
	if len(falconSensor.Tags) > 0 {
		sensorConfig["FALCONCTL_OPT_TAGS"] = strings.Join(falconSensor.Tags, ",")
	}
	if falconSensor.Trace != "" {
		sensorConfig["FALCONCTL_OPT_TRACE"] = falconSensor.Trace
	}
	return sensorConfig
}
