package common

const (
	FalconContainerInjection                = "sensor.falcon-system.crowdstrike.com/injection"
	FalconContainerInjectorTLSName          = "injector-tls"
	FalconHostInstallDir                    = "/opt"
	FalconInitHostInstallDir                = "/host_opt"
	FalconDataDir                           = "/opt/CrowdStrike"
	FalconInitDataDir                       = "/host_opt/CrowdStrike/"
	FalconConfigDir                         = "/opt/CrowdStrike/config"
	FalconStoreFile                         = "/opt/CrowdStrike/falconstore"
	FalconInitStoreFile                     = "/host_opt/CrowdStrike/falconstore"
	FalconDaemonsetInitBinary               = "/opt/CrowdStrike/falcon-daemonset-init -i"
	FalconDaemonsetConfigureClusterIdBinary = "/opt/CrowdStrike/configure-cluster-id"
	FalconDaemonsetCleanupBinary            = "/opt/CrowdStrike/falcon-daemonset-init -u"
	FalconDaemonsetBinary                   = "/opt/CrowdStrike/falcon-daemonset-init"
	FalconContainerProbePath                = "/live"
	FalconAdmissionClientStartupProbePath   = "/startz"
	FalconAdmissionClientLivenessProbePath  = "/livez"
	FalconAdmissionStartupProbePath         = "/startz-kac"
	FalconAdmissionLivenessProbePath        = "/livez-kac"
	FalconAdmissionServiceHTTPSName         = "webhook-port"
	FalconServiceHTTPSName                  = "https"
	FalconServiceHTTPSPort                  = 443
	FalconAdmissionWatcherPortName          = "watcher-health"
	FalconAdmissionWatcherPort              = int32(4080)
	FalconAdmissionValidatingWebhookName    = "validating.admission.falcon.crowdstrike.com"
	FalconAdmissionClusterNameConfigMapName = "falcon-kac-meta"
	FalconAdmissionComponentName            = "kac"
	FalconAdmissionServiceApp               = "falcon-kac"
	FalconImageAnalyzerComponentName        = "iar"
	FalconImageAnalyzerAgentService         = "iar-agent-service"
	FalconImageAnalyzerAgentServiceApp      = "falcon-image-analyzer"
	FalconImageAnalyzerAgentServicePort     = 443
	FalconImageAnalyzerAgentServicePortName = "service-port"

	AppLabelKey              = "app"
	KubernetesComponentKey   = "app.kubernetes.io/component"
	KubernetesNameKey        = "app.kubernetes.io/name"
	FalconInstanceNameKey    = "crowdstrike.com/name"
	FalconInstanceKey        = "crowdstrike.com/instance"
	FalconComponentKey       = "crowdstrike.com/component"
	FalconManagedByKey       = "crowdstrike.com/managed-by"
	FalconPartOfKey          = "crowdstrike.com/part-of"
	FalconProviderKey        = "crowdstrike.com/provider"
	FalconCreatedKey         = "crowdstrike.com/created-by"
	FalconAdmissionReviewKey = "falcon.crowdstrike.com/admission-review"

	FalconOperatorVersionKey = "crowdstrike.com/operator-version"

	FalconKernelSensor        = "kernel_sensor"
	FalconSidecarSensor       = "container_sensor"
	FalconAdmissionController = "admission_controller"
	FalconImageAnalyzer       = "falcon-imageanalyzer"
	FalconClusterGuard        = "falcon-clusterguard"
	FalconFinalizer           = "falcon.crowdstrike.com/finalizer"
	FalconProviderValue       = "crowdstrike"
	FalconPartOfValue         = "Falcon"
	FalconCreatedValue        = "falcon-operator"
	FalconManagedByValue      = "controller-manager"
	FalconPriorityClassName   = "system-cluster-critical"

	SidecarServiceAccountName   = "falcon-operator-sidecar-sensor"
	FalconPullSecretName        = "crowdstrike-falcon-pull-secret"
	NodeServiceAccountName      = "falcon-operator-node-sensor"
	AdmissionServiceAccountName = "falcon-operator-admission-controller"
	NodeClusterRoleBindingName  = "falcon-operator-node-sensor-rolebinding"
	ImageServiceAccountName     = "falcon-operator-image-analyzer"

	ClusterGuardSensorConfigMapName             = "falcon-sensor-config"
	ClusterGuardSensorPriorityClassName         = "falcon-sensor-priorityclass"
	ClusterGuardSensorClusterRoleBindingName    = "falcon-sensor-access-binding"
	ClusterGuardSensorDaemonSetName             = "falcon-sensor"
	ClusterGuardSensorCleanupServiceAccountName = "falcon-node-sensor-sa-node-cleanup"
	ClusterGuardSensorCleanupDaemonSetName      = "falcon-sensor-node-cleanup"

	// ClusterRoles and Roles created by kustomize
	AdmissionClusterRoleName = "falcon-operator-admission-controller-role"
	NodeClusterRoleName      = "falcon-operator-node-sensor-role"

	// Admission Sensor Module Vars
	AdmissionNamespaceRoleName          = "falcon-clusterguard-namespace-role"
	AdmissionModuleServiceAccountName   = "falcon-cg-controller-sa"
	AdmissionDeploymentName             = "falcon-clusterguard-controller"
	AdmissionConfigMapName              = "falcon-clusterguard-config"
	AdmissionClusterRoleBindingName     = "falcon-clusterguard-security-crb"
	AdmissionRoleName                   = "falcon-clusterguard-role"
	AdmissionRoleBindingName            = "falcon-clusterguard-rolebinding"
	AdmissionResourceQuotaName          = "falcon-clusterguard-quota"
	AdmissionServiceApp                 = "falcon-kac"
	ClusterGuardServiceApp              = "falcon-clusterguard"
	AdmissionWebhookServiceName         = "webhook"
	AdmissionAPIServiceName             = "api"
	AdmissionWebhookPort                = int32(4443)
	AdmissionWebhookPortStr             = "4443"
	AdmissionGRPCPort                   = int32(50051)
	AdmissionGRPCPortStr                = "50051"
	AdmissionWatcherHTTPPort            = int32(4080)
	AdmissionWatcherHTTPPortStr         = "4080"
	AdmissionTLSSecretName              = "falcon-cg-controller-tls"
	AdmissionAPITLSSecretName           = "falcon-api-tls"
	AdmissionAPICASecretName            = "falcon-api-ca"
	ClusterGuardSensorTLSSecretName     = "falcon-sensor-tls"
	ClusterGuardNodeSensorTLSSecretName = "falcon-node-sensor-tls"
	AdmissionValidatingWebhookName      = "validating.falcon-clusterguard.crowdstrike.com"
	AdmissionReviewLabelKey             = "falcon-clusterguard.crowdstrike.com/admission-review"
	AdmissionComponentName              = "kac"

	// Node Sensor Module Vars
	ClusterGuardSensorServiceAccountName = "falcon-node-sensor-sa"
	ClusterGuardProxyServiceName         = "falcon-proxy"
	ClusterGuardProxyPortName            = "proxy"
	ClusterGuardProxyTLSVolumeName       = "proxy-tls"
	ClusterGuardProxyServicePort         = int32(80)
	ClusterGuardProxyMountPath           = "/run/secrets/proxy"

	// Shared between Admission and Node Sensor modules
	ClusterGuardComponentName           = "falcon-clusterguard"
	ClusterGuardNodeSensorComponentName = "node_sensor"
	ClusterGuardAPIServiceName          = "api"
	ClusterGuardAPICASecretName         = "falcon-api-ca"

	// FCG component ClusterRole names (installed by kustomize, referenced by component CRBs)
	FCGAdmissionClusterRoleName       = "falcon-operator-falcon-clusterguard-resource-reader"
	ClusterGuardSensorClusterRoleName = "falcon-operator-falcon-sensor-access-role"

	// GKE Autopilot requires names to have an exact match for WorkloadAllowlists
	GKEAutoPilotConfigMapName           = "falcon-node-sensor-config"
	GKEAutoPilotAllowListLabelKey       = "cloud.google.com/matching-allowlist"
	GKEAutoPilotDeployDSAllowlistPrefix = "crowdstrike-falconsensor-deploy-allowlist"
	GKEAutoPilotCleanupAllowlistPrefix  = "crowdstrike-falconsensor-cleanup-allowlist"

	// Deprecation Variables
	FalconAdmissionEnabled  = false
	FalconNodeSensorEnabled = false
)
