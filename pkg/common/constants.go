package common

const (
	FalconContainerInjection = "sensor.falcon-system.crowdstrike.com/injection"
	FalconDataDir            = "/var/lib/crowdstrike"
	FalconStoreFile          = "/var/lib/crowdstrike/falconstore"
	FalconDefaultImage       = "falcon-node-sensor:latest"
	FalconContainerProbePath = "/live"
	FalconServiceHTTPSName   = "https"
	FalconServiceHTTPSPort   = 443

	FalconInstanceNameKey = "crowdstrike.com/name"
	FalconInstanceKey     = "crowdstrike.com/instance"
	FalconComponentKey    = "crowdstrike.com/component"
	FalconManagedByKey    = "crowdstrike.com/managed-by"
	FalconPartOfKey       = "crowdstrike.com/part-of"
	FalconProviderKey     = "crowdstrike.com/provider"
	FalconControllerKey   = "crowdstrike.com/created-by"
)
