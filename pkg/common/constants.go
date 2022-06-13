package common

const (
	FalconContainerInjection = "sensor.falcon-system.crowdstrike.com/injection"
	FalconHostInstallDir     = "/opt"
	FalconDataDir            = "/opt/CrowdStrike"
	FalconStoreFile          = "/opt/CrowdStrike/falconstore"
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
	FalconKernelSensor    = "kernel_sensor"

	FalconFinalizer     = "falcon.crowdstrike.com/finalizer"
	FalconProviderValue = "crowdstrike"

	FalconInstallerJobContainerName = "installer"

	FalconPullSecretName       = "crowdstrike-falcon-pull-secret"
	NodeServiceAccountName     = "crowdstrike-falcon-node-sensor"
	NodeClusterRoleName        = "crowdstrike-falcon-node-sensor"
	NodeClusterRoleBindingName = "crowdstrike-falcon-node-sensor"
	NodeSccName                = "crowdstrike-falcon-node-sensor"
)
