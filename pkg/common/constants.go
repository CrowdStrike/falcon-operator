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

	// ClusterGuard Controller Module Vars
	ClusterGuardControllerNamespaceRoleName      = "falcon-clusterguard-namespace-role"
	ClusterGuardControllerServiceAccountName     = "falcon-cg-controller-sa"
	ClusterGuardControllerDeploymentName         = "falcon-clusterguard-controller"
	ClusterGuardControllerConfigMapName          = "falcon-clusterguard-config"
	ClusterGuardControllerClusterRoleBindingName = "falcon-clusterguard-security-crb"
	ClusterGuardControllerRoleName               = "falcon-clusterguard-role"
	ClusterGuardControllerRoleBindingName        = "falcon-clusterguard-rolebinding"
	ClusterGuardControllerResourceQuotaName      = "falcon-clusterguard-quota"
	ClusterGuardControllerServiceApp             = "falcon-kac"
	ClusterGuardServiceApp                       = "falcon-clusterguard"
	ClusterGuardControllerWebhookServiceName     = "webhook"
	ClusterGuardControllerAPIServiceName         = "api"
	ClusterGuardControllerWebhookPort            = int32(4443)
	ClusterGuardControllerWebhookPortStr         = "4443"
	ClusterGuardControllerGRPCPort               = int32(50051)
	ClusterGuardControllerGRPCPortStr            = "50051"
	ClusterGuardControllerWatcherHTTPPort        = int32(4080)
	ClusterGuardControllerWatcherHTTPPortStr     = "4080"
	ClusterGuardControllerTLSSecretName          = "falcon-cg-controller-tls"
	ClusterGuardControllerAPITLSSecretName       = "falcon-api-tls"
	ClusterGuardControllerAPICASecretName        = "falcon-api-ca"
	ClusterGuardControllerValidatingWebhookName  = "validating.falcon-clusterguard.crowdstrike.com"
	ClusterGuardControllerReviewLabelKey         = "falcon-clusterguard.crowdstrike.com/admission-review"
	ClusterGuardControllerComponentName          = "kac"

	// Node Sensor Module Vars
	ClusterGuardNodeSensorServiceAccountName        = "falcon-node-sensor-sa"
	ClusterGuardNodeSensorConfigMapName             = "falcon-sensor-config"
	ClusterGuardNodeSensorPriorityClassName         = "falcon-sensor-priorityclass"
	ClusterGuardNodeSensorClusterRoleBindingName    = "falcon-sensor-access-binding"
	ClusterGuardNodeSensorCleanupServiceAccountName = "falcon-node-sensor-sa-node-cleanup"
	ClusterGuardNodeSensorTLSSecretName             = "falcon-node-sensor-tls"

	// Shared between Cluster Guard Controller and Node Sensor modules
	ClusterGuardComponentName           = "falcon-clusterguard"
	ClusterGuardNodeSensorComponentName = "node_sensor"
	ClusterGuardAPIServiceName          = "api"
	ClusterGuardAPICASecretName         = "falcon-api-ca"

	// FCG component ClusterRole names (installed by kustomize, referenced by component CRBs)
	ClusterGuardControllerClusterRoleName  = "falcon-operator-falcon-clusterguard-resource-reader"
	ClusterGuardNodeSensorClusterRoleName  = "falcon-operator-falcon-sensor-access-role"

	// GKE Autopilot requires names to have an exact match for WorkloadAllowlists
	GKEAutoPilotConfigMapName           = "falcon-node-sensor-config"
	GKEAutoPilotAllowListLabelKey       = "cloud.google.com/matching-allowlist"
	GKEAutoPilotDeployDSAllowlistPrefix = "crowdstrike-falconsensor-deploy-allowlist"
	GKEAutoPilotCleanupAllowlistPrefix  = "crowdstrike-falconsensor-cleanup-allowlist"

	// Deprecation Variables
	FalconAdmissionEnabled  = false
	FalconNodeSensorEnabled = false
)
