package common

const (
	FalconContainerInjection               = "sensor.falcon-system.crowdstrike.com/injection"
	FalconContainerInjectorTLSName         = "injector-tls"
	FalconHostInstallDir                   = "/opt"
	FalconInitHostInstallDir               = "/host_opt"
	FalconDataDir                          = "/opt/CrowdStrike"
	FalconInitDataDir                      = "/host_opt/CrowdStrike/"
	FalconStoreFile                        = "/opt/CrowdStrike/falconstore"
	FalconInitStoreFile                    = "/host_opt/CrowdStrike/falconstore"
	FalconDaemonsetInitBinary              = "/opt/CrowdStrike/falcon-daemonset-init"
	FalconDaemonsetInitBinaryInvocation    = "falcon-daemonset-init -i"
	FalconDaemonsetCleanupBinaryInvocation = "falcon-daemonset-init -u"
	FalconContainerProbePath               = "/live"
	FalconServiceHTTPSName                 = "https"
	FalconServiceHTTPSPort                 = 443

	FalconInstanceNameKey    = "crowdstrike.com/name"
	FalconInstanceKey        = "crowdstrike.com/instance"
	FalconComponentKey       = "crowdstrike.com/component"
	FalconManagedByKey       = "crowdstrike.com/managed-by"
	FalconPartOfKey          = "crowdstrike.com/part-of"
	FalconProviderKey        = "crowdstrike.com/provider"
	FalconCreatedKey         = "crowdstrike.com/created-by"
	FalconAdmissionReviewKey = "falcon.crowdstrike.com/admission-review"

	FalconKernelSensor        = "kernel_sensor"
	FalconSidecarSensor       = "container_sensor"
	FalconAdmissionController = "admission_controller"
	FalconFinalizer           = "falcon.crowdstrike.com/finalizer"
	FalconProviderValue       = "crowdstrike"
	FalconPartOfValue         = "Falcon"
	FalconCreatedValue        = "falcon-operator"
	FalconManagedByValue      = "controller-manager"

	SidecarServiceAccountName  = "falcon-operator-sidecar-sensor"
	FalconPullSecretName       = "crowdstrike-falcon-pull-secret"
	NodeServiceAccountName     = "falcon-operator-node-sensor"
	NodeClusterRoleBindingName = "crowdstrike-falcon-node-sensor"
)
