package falcon_api

import (
	"fmt"
	"github.com/crowdstrike/gofalcon/falcon/client/falcon_container"
	"github.com/crowdstrike/gofalcon/falcon/client/oauth2"
	"github.com/crowdstrike/gofalcon/falcon/client/sensor_download"
)

func errorHint(err error, extraDescription string) error {
	switch e := err.(type) {
	case *falcon_container.GetCredentialsForbidden:
		return fmt.Errorf("Insufficient CrowdStrike privileges, please grant [Falcon Images Download: Read] to CrowdStrike API Key. Error was: %s", err)
	case *sensor_download.GetSensorInstallersCCIDByQueryForbidden:
		return fmt.Errorf("Insufficient CrowdStrike privileges, please grant [Sensor Download: Read] to CrowdStrike API Key. Error was: %s", err)
	case *oauth2.Oauth2AccessTokenForbidden:
		if e.Payload != nil && len(e.Payload.Errors) == 1 && e.Payload.Errors[0] != nil && e.Payload.Errors[0].Message != nil && *e.Payload.Errors[0].Message == "access denied, authorization failed" {
			return fmt.Errorf("Please check the settings of IP-based allowlisting in CrowdStrike Falcon Console. %s", e)
		}
	}
	if extraDescription != "" {
		return fmt.Errorf("%s. Error was: %s", extraDescription, err)
	} else {
		return err
	}
}
