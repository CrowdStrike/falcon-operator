package common

const (
	FalconContainerInjection       = "sensor.falcon-system.crowdstrike.com/injection"
	FalconContainerInjectorTLSName = "injector-tls"
	FalconHostInstallDir           = "/opt"
	FalconDataDir                  = "/opt/CrowdStrike"
	FalconStoreFile                = "/opt/CrowdStrike/falconstore"
	FalconContainerProbePath       = "/live"
	FalconServiceHTTPSName         = "https"
	FalconServiceHTTPSPort         = 443

	FalconInstanceNameKey = "crowdstrike.com/name"
	FalconInstanceKey     = "crowdstrike.com/instance"
	FalconComponentKey    = "crowdstrike.com/component"
	FalconManagedByKey    = "crowdstrike.com/managed-by"
	FalconPartOfKey       = "crowdstrike.com/part-of"
	FalconProviderKey     = "crowdstrike.com/provider"
	FalconCreatedKey      = "crowdstrike.com/created-by"

	FalconKernelSensor   = "kernel_sensor"
	FalconSidecarSensor  = "container_sensor"
	FalconFinalizer      = "falcon.crowdstrike.com/finalizer"
	FalconProviderValue  = "crowdstrike"
	FalconPartOfValue    = "Falcon"
	FalconCreatedValue   = "falcon-operator"
	FalconManagedByValue = "controller-manager"

	FalconPullSecretName       = "crowdstrike-falcon-pull-secret"
	NodeServiceAccountName     = "crowdstrike-falcon-node-sensor"
	NodeClusterRoleName        = "crowdstrike-falcon-node-sensor"
	NodeClusterRoleBindingName = "crowdstrike-falcon-node-sensor"
	NodeSccName                = "crowdstrike-falcon-node-sensor"
)
