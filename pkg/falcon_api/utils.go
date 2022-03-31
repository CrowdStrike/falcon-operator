package falcon_api

import (
	"fmt"
	"github.com/crowdstrike/gofalcon/falcon/client/falcon_container"
	"github.com/crowdstrike/gofalcon/falcon/client/sensor_download"
)

func errorHint(err error, extraDescription string) error {
	switch err.(type) {
	case *falcon_container.GetCredentialsForbidden:
		return fmt.Errorf("Insufficient CrowdStrike privileges, please grant [Falcon Images Download: Read] to CrowdStrike API Key. Error was: %s", err)
	case *sensor_download.GetSensorInstallersCCIDByQueryForbidden:
		return fmt.Errorf("Insufficient CrowdStrike privileges, please grant [Sensor Download: Read] to CrowdStrike API Key. Error was: %s", err)
	}
	if extraDescription != "" {
		return fmt.Errorf("%s. Error was: %s", extraDescription, err)
	} else {
		return err
	}
}
