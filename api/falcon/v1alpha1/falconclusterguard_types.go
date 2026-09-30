package v1alpha1

import (
	"time"

	arv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	ClusterGuardAdmissionControlEnabledDefault = true
	ClusterGuardWatchEventsEnabledDefault      = true
	ClusterGuardSnapshotsEnabledDefault        = true
	ClusterGuardSnapshotIntervalDefault        = 22
	ClusterGuardWatcherPortDefault             = int32(4080)

	DisableClusterGuardControllerAck = "I understand this disables K8s metadata collection and degrades node sensor visibility"
)

// FalconClusterGuardRegistryType specifies the type of registry used for Cloud Guard images
type FalconClusterGuardRegistryType string

const (
	// RegistryTypeFalconClusterGuardCrowdStrike uses the CrowdStrike registry (default)
	RegistryTypeFalconClusterGuardCrowdStrike FalconClusterGuardRegistryType = "crowdstrike"
	// RegistryTypeFalconClusterGuardPrivate uses a private registry configured via ImagePullSecrets or cloud-specific settings
	RegistryTypeFalconClusterGuardPrivate FalconClusterGuardRegistryType = "private"
)

// FalconClusterGuardRegistrySpec configures the registry source for Cloud Guard images
type FalconClusterGuardRegistrySpec struct {
	// Type specifies whether to use CrowdStrike's registry or a private registry
	// +kubebuilder:default=crowdstrike
	// +kubebuilder:validation:Enum=crowdstrike;private
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Registry Type",order=1
	Type FalconClusterGuardRegistryType `json:"type,omitempty"`

	// TLS configures TLS settings for connecting to the container image registry
	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Registry TLS Configuration",order=2
	TLS RegistryTLSSpec `json:"tls,omitempty"`
}

// FalconClusterGuardNodeSpec defines configuration for the node sensor DaemonSet deployed by FalconClusterGuard.
type FalconClusterGuardNodeSpec struct {
	// Enabled controls whether the node sensor DaemonSet is deployed. Defaults to true.
	// +kubebuilder:default:=true
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Enable Node Sensor",order=1
	Enabled *bool `json:"enabled,omitempty"`

	// Specifies tolerations for custom taints. Defaults to allowing scheduling on all nodes.
	// +optional
	// +kubebuilder:default:={{key: "node-role.kubernetes.io/master", operator: "Exists", effect: "NoSchedule"}, {key: "node-role.kubernetes.io/control-plane", operator: "Exists", effect: "NoSchedule"}, {key: "node-role.kubernetes.io/infra", operator: "Exists", effect: "NoSchedule"}, {key: "kubernetes.azure.com/scalesetpriority", operator: "Equal", value: "spot", effect: "NoSchedule"}}
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=4
	Tolerations *[]corev1.Toleration `json:"tolerations"`

	// Specifies node affinity for scheduling the DaemonSet. Defaults to allowing scheduling on all nodes.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=5
	NodeAffinity corev1.NodeAffinity `json:"nodeAffinity,omitempty"`

	// Type of DaemonSet update. Can be "RollingUpdate" or "OnDelete". Default is RollingUpdate.
	// +kubebuilder:default={}
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="DaemonSet Update Strategy",order=6
	DSUpdateStrategy FalconClusterGuardUpdateStrategy `json:"updateStrategy,omitempty"`

	// Kills pod after a specificed amount of time (in seconds). Default is 60 seconds.
	// +kubebuilder:default:=60
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=7
	TerminationGracePeriod *int64 `json:"terminationGracePeriod,omitempty"`

	// Add metadata to the DaemonSet Service Account for IAM roles.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	ServiceAccount FalconClusterGuardNodeServiceAccount `json:"serviceAccount,omitempty"`

	// Disables the cleanup of the sensor through DaemonSet on the nodes.
	// Disabling might have unintended consequences for certain operations such as sensor downgrading.
	// +kubebuilder:default=false
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=8
	NodeCleanup *bool `json:"disableCleanup,omitempty"`

	// Configure resource requests and limits for the DaemonSet Sensor. Only applies when using the eBPF backend.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon eBPF Sensor Resources",order=9
	SensorResources FalconClusterGuardResources `json:"resources,omitempty"`

	// Sets the backend to be used by the DaemonSet Sensor.
	// +kubebuilder:default=bpf
	// +kubebuilder:validation:Enum=kernel;bpf
	// +operator-sdk-csv:customresourcedefinitions:type=spec,order=10
	Backend string `json:"backend,omitempty"`

	// Enables the use of GKE Autopilot.
	// +kubebuilder:default={}
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="GKE Autopilot Settings",order=11
	GKE FalconClusterGuardAutoPilot `json:"gke,omitempty"`

	// Enable priority class for the DaemonSet. This is useful for GKE Autopilot clusters, but can be set for any cluster.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Priority Class",order=12
	PriorityClass FalconClusterGuardPriorityClassConfig `json:"priorityClass,omitempty"`

	// Version of the sensor to be installed. The latest version will be selected when this version specifier is missing.
	Version *string `json:"version,omitempty"`

	// Advanced configures various options that go against industry practices or are otherwise not recommended for use.
	// Adjusting these settings may result in incorrect or undesirable behavior. Proceed at your own risk.
	// For more information, please see https://github.com/CrowdStrike/falcon-operator/blob/main/docs/ADVANCED.md.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="DaemonSet Advanced Settings"
	Advanced FalconAdvanced `json:"advanced,omitempty"`

	// When running on an unmanaged K8S cluster, set a cluster name. When running on managed, K8S cluster name is resolved cloud-side
	// +kubebuilder:validation:Pattern="^[0-9a-zA-Z]{1}[0-9a-zA-Z_-]{1,99}$"
	ClusterName *string `json:"clusterName,omitempty"`
}

// FalconClusterGuardController defines configuration for the admission controller deployed by FalconClusterGuard.
type FalconClusterGuardController struct {
	// To prevent the Falcon Cluster Guard Controller from being deployed, set this field to:
	// "I understand this disables K8s metadata collection and degrades node sensor visibility"
	// WARNING: Disabling the controller removes Kubernetes metadata collection and degrades node sensor visibility.
	// +kubebuilder:validation:Enum="I understand this disables K8s metadata collection and degrades node sensor visibility"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Disable Falcon Cluster Guard Controller",order=1
	// +optional
	DisableClusterGuardController string `json:"disableClusterGuardController,omitempty"`

	// Define annotations that will be passed down to Falcon Cluster Guard Controller  service account. This is useful for passing along AWS IAM Role or GCP Workload Identity.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Service Account Configuration",order=7
	ServiceAccount FalconClusterGuardControllerServiceAccount `json:"serviceAccount,omitempty"`

	// Port on which the Falcon Cluster Guard Controller service will listen for requests from the cluster.
	// +kubebuilder:default:=443
	// +kubebuilder:validation:XIntOrString
	// +kubebuilder:validation:Minimum:=0
	// +kubebuilder:validation:Maximum:=65535
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Cluster Guard Controller Service Port",order=3,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number"}
	Port *int32 `json:"servicePort,omitempty"`

	// Port on which the Falcon Cluster Guard Controller container will listen for requests.
	// +kubebuilder:default:=4443
	// +kubebuilder:validation:XIntOrString
	// +kubebuilder:validation:Minimum:=0
	// +kubebuilder:validation:Maximum:=65535
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Cluster Guard Controller Container Port",order=4,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number"}
	ContainerPort *int32 `json:"containerPort,omitempty"`

	// Port on which the Falcon Cluster Guard Watcher container will listen for health probes.
	// +kubebuilder:default:=4080
	// +kubebuilder:validation:XIntOrString
	// +kubebuilder:validation:Minimum:=0
	// +kubebuilder:validation:Maximum:=65535
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Cluster Guard Watcher HTTP Port",order=5,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number"}
	WatcherPort *int32 `json:"watcherPort,omitempty"`

	// Configure TLS settings for the Falcon Cluster Guard Controller Controller
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Cluster Guard Controller TLS Configuration",order=8
	TLS FalconClusterGuardControllerTLS `json:"tls,omitempty"`

	// Configure the failure policy for the Falcon Cluster Guard Controller Admission Controller.
	// +kubebuilder:default:=Ignore
	// +kubebuilder:validation:Enum=Ignore;Fail
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Cluster Guard Controller Admission Controller Failure Policy",order=6
	FailurePolicy arv1.FailurePolicyType `json:"failurePolicy,omitempty"`

	// Ignore admission control for a specific set of namespaces.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Ignore Namespace List",order=12
	DisabledNamespaces FalconClusterGuardControllerNamespace `json:"disabledNamespaces,omitempty"`

	// Determines if Kubernetes resources are watched for cluster visibility.
	// +kubebuilder:default:=true
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Enable Resource Watcher",order=14
	WatcherEnabled *bool `json:"watcherEnabled,omitempty"`

	// Determines if snapshots of Kubernetes resources are periodically taken for cluster visibility.
	// +kubebuilder:default:=true
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Enable Resource Snapshots",order=15
	SnapshotsEnabled *bool `json:"snapshotsEnabled,omitempty"`

	// Time interval between two snapshots of Kubernetes resources in the cluster.
	// +kubebuilder:default:="22h"
	// +kubebuilder:validation:Type:=string
	// +kubebuilder:validation:Format:=duration
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Time Interval Between Two Snapshots",order=16
	SnapshotsInterval *metav1.Duration `json:"snapshotsInterval,omitempty"`

	// Determines if the admission controller webhook is enabled.
	// +kubebuilder:default:=false
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Enable Admission Controller",order=18
	AdmissionControlEnabled *bool `json:"admissionControlEnabled,omitempty"`

	// Determines if the admission controller watches for configMap events.
	// +kubebuilder:default:=true
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Enable ConfigMap Event Watcher",order=17
	ConfigMapWatcherEnabled *bool `json:"configMapWatcherEnabled,omitempty"`

	// Namespace where Falcon Image Analyzer is installed. Falcon Cluster Guard needs to know this to discover and communicate with IAR.
	// +kubebuilder:default:="falcon-iar"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Image Analyzer Namespace",order=20
	FalconImageAnalyzerNamespace *string `json:"falconImageAnalyzerNamespace,omitempty"`

	// Currently ignored and internally set to 1.
	// +kubebuilder:default:=2
	// +kubebuilder:validation:XIntOrString
	// +kubebuilder:validation:Minimum:=0
	// +kubebuilder:validation:Maximum:=65535
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Cluster Guard Controller Replica Count",order=5,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number"}
	Replicas *int32 `json:"replicas,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Cluster Guard Controller Client Resources",order=9,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:resourceRequirements"}
	// +kubebuilder:default:={"limits":{"memory":"384Mi"},"requests":{"cpu":"250m","memory":"384Mi"}}
	ResourcesClient *corev1.ResourceRequirements `json:"resourcesClient,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Cluster Guard Controller Client Resources",order=9,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:resourceRequirements"}
	// +kubebuilder:default:={"limits":{"memory":"128Mi"},"requests":{"cpu":"100m","memory":"128Mi"}}
	ResourcesClientNoWebhook *corev1.ResourceRequirements `json:"resourcesClientNoWebhook,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Cluster Guard Controller Watcher Resources",order=18,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:resourceRequirements"}
	// +kubebuilder:default:={"limits":{"memory":"384Mi"},"requests":{"cpu":"250m","memory":"384Mi"}}
	ResourcesWatcher *corev1.ResourceRequirements `json:"resourcesWatcher,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Cluster Guard Controller Admission Controller Resources",order=10,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:resourceRequirements"}
	// +kubebuilder:default:={"limits":{"memory":"256Mi"},"requests":{"cpu":"100m","memory":"256Mi"}}
	ResourcesAC *corev1.ResourceRequirements `json:"resources,omitempty"`

	// Type of Deployment update. Can be "RollingUpdate" or "OnDelete". Default is RollingUpdate.
	// +kubebuilder:default:={}
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Deployment Update Strategy",order=11
	DepUpdateStrategy FalconClusterGuardControllerUpdateStrategy `json:"updateStrategy,omitempty"`

	// Specifies node affinity for scheduling the Falcon Cluster Guard Controller.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=19
	NodeAffinity *corev1.NodeAffinity `json:"nodeAffinity,omitempty"`

	// Specifies tolerations for scheduling the Falcon Cluster Guard Controller.
	// +kubebuilder:default:={}
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=20
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// Cluster Name if Falcon Cluster Guard cannot discover the cluster name. This will be overwritten if Falcon Cluster Guard is able to discover the cluster name.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Admission Cluster Name",order=21
	ClusterName *string `json:"clusterName,omitempty"`

	// Minimum TLS version accepted by the webhook server. Valid values are "TLS1.2" and "TLS1.3".
	// When unset the server default is used.
	// +kubebuilder:validation:Enum=TLS1.2;TLS1.3
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Minimum TLS Version",order=22
	// +optional
	TLSVersionMinimum *string `json:"tlsVersionMinimum,omitempty"`

	// Maximum number of pods allowed with the system-cluster-critical priority class.
	// Controls the ResourceQuota for the admission controller namespace. Defaults to 2.
	// +kubebuilder:default:=2
	// +kubebuilder:validation:Minimum:=0
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Resource Quota Pods",order=23
	// +optional
	ResourceQuotaPods *int32 `json:"resourceQuotaPods,omitempty"`
}

func (s *FalconClusterGuardNodeSpec) IsEnabled() bool {
	return s.Enabled == nil || *s.Enabled
}

func (s *FalconClusterGuardController) IsEnabled() bool {
	return s.DisableClusterGuardController != DisableClusterGuardControllerAck
}

func (s *FalconClusterGuardController) GetWatcherEnabled() bool {
	if s.WatcherEnabled == nil {
		return WatcherEnabledDefault
	}
	return *s.WatcherEnabled
}

func (s *FalconClusterGuardController) GetSnapshotsEnabled() bool {
	if s.SnapshotsEnabled == nil {
		return SnapshotsEnabledDefault
	}
	return *s.SnapshotsEnabled
}

func (s *FalconClusterGuardController) GetSnapshotsInterval() time.Duration {
	if s.SnapshotsInterval == nil {
		return SnapshotsIntervalDefault * time.Hour
	}
	return s.SnapshotsInterval.Duration
}

func (s *FalconClusterGuardController) GetWatcherPort() int32 {
	if s.WatcherPort == nil {
		return ClusterGuardWatcherPortDefault
	}
	return *s.WatcherPort
}

func (s *FalconClusterGuardController) GetConfigMapWatcherEnabled() bool {
	if s.ConfigMapWatcherEnabled == nil {
		return ConfigMapWatcherEnabledDefault
	}
	return *s.ConfigMapWatcherEnabled
}

// FalconClusterGuardSpec defines the desired state of FalconClusterGuard
type FalconClusterGuardSpec struct {
	// InstallNamespace is the namespace where the Falcon Cloud Guard resources will be deployed
	// +kubebuilder:default:=falcon-system
	// +optional
	InstallNamespace string `json:"installNamespace,omitempty"`

	// CrowdStrike Falcon sensor configuration
	// +kubebuilder:default:={}
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Sensor Configuration",order=1
	Falcon FalconSensor `json:"falcon,omitempty"`

	// FalconAPI configures connection from your local Falcon operator to CrowdStrike Falcon platform.
	//
	// When configured, it will pull the sensor from registry.crowdstrike.com and deploy the appropriate sensor to the cluster.
	//
	// If using the API is not desired, the sensor can be manually configured by setting the Image field.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Platform API Configuration",order=2
	// +optional
	FalconAPI *FalconAPI `json:"falcon_api,omitempty"`

	// FalconSecret config is used to inject k8s secrets with sensitive data for the FalconAPI.
	// The following Falcon values are supported by k8s secret injection:
	//   falcon-client-id
	//   falcon-client-secret
	//   falcon-cid
	//   falcon-provisioning-token
	// +kubebuilder:default={"enabled": false}
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Platform Secrets Configuration",order=2
	FalconSecret FalconSecret `json:"falconSecret,omitempty"`

	// Registry configures the container image registry used for the Cloud Guard image.
	// +kubebuilder:default={"type":"crowdstrike"}
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Cloud Guard Registry Configuration",order=3
	// +optional
	Registry FalconClusterGuardRegistrySpec `json:"registry,omitempty"`

	// Location of the Falcon sensor image. Used for all deployed components. Use only when mirroring the image to a custom repository.
	// +kubebuilder:validation:Pattern="^.*:.*$"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=4
	Image string `json:"image,omitempty"`

	// +kubebuilder:default=IfNotPresent
	// +kubebuilder:validation:Enum=Always;IfNotPresent;Never
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=5
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// ImagePullSecrets is an optional list of references to secrets used for pulling images across all deployed components.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=6
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// Version of the sensor to be installed. The latest version will be selected when this version specifier is missing.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Sensor Version",order=7
	// +optional
	Version *string `json:"version,omitempty"`

	// ClusterGuardControllerConfig configures the controller deployed alongside FalconClusterGuard.
	// +kubebuilder:default={}
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Cluster Guard Controller Configuration",order=8
	ClusterGuardControllerConfig FalconClusterGuardController `json:"controller,omitempty"`

	// NodeSensor configures the node sensor DaemonSet deployed alongside FalconClusterGuard.
	// +kubebuilder:default={}
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Node Sensor Configuration",order=9
	NodeSensor FalconClusterGuardNodeSpec `json:"nodeSensor,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:scope=Cluster,shortName=fcg,categories={falcon}
// +kubebuilder:printcolumn:name="Sensor",type="string",JSONPath=".status.sensor",description="Version of the Falcon Sensor"
// +kubebuilder:printcolumn:name="Node Sensor",type="string",JSONPath=".spec.nodeSensor.enabled",description="Node sensor component enabled"
// +kubebuilder:printcolumn:name="Controller",type="string",JSONPath=".spec.controller.disableClusterGuardController",description="Cluster Guard Controller disabled when set"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.conditions[?(@.type=='Success')].reason",description="Deployment status"
// +kubebuilder:printcolumn:name="Operator Version",type="string",JSONPath=".status.version",description="Version of the operator",priority=1
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp",description="Age of the resource"

// FalconClusterGuard is the Schema for the falconclusterguards API
type FalconClusterGuard struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitzero"`

	// spec defines the desired state of FalconClusterGuard
	// +required
	Spec FalconClusterGuardSpec `json:"spec"`

	// status defines the observed state of FalconClusterGuard
	// +optional
	Status FalconCRStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// FalconClusterGuardList contains a list of FalconClusterGuard
type FalconClusterGuardList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitzero"`
	Items           []FalconClusterGuard `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FalconClusterGuard{}, &FalconClusterGuardList{})
}

// FalconCRD interface implementation

// GetFalconSecretSpec returns the FalconSecret configuration
func (f *FalconClusterGuard) GetFalconSecretSpec() FalconSecret {
	return f.Spec.FalconSecret
}

// GetFalconAPISpec returns the FalconAPI configuration
func (f *FalconClusterGuard) GetFalconAPISpec() *FalconAPI {
	return f.Spec.FalconAPI
}

// SetFalconAPISpec sets the FalconAPI configuration
func (f *FalconClusterGuard) SetFalconAPISpec(api *FalconAPI) {
	f.Spec.FalconAPI = api
}

// GetFalconSpec returns the FalconSensor configuration
func (f *FalconClusterGuard) GetFalconSpec() FalconSensor {
	return f.Spec.Falcon
}

// SetFalconSpec sets the FalconSensor configuration
func (f *FalconClusterGuard) SetFalconSpec(sensor FalconSensor) {
	f.Spec.Falcon = sensor
}

// GetClusterName returns the cluster name from controller.
func (f *FalconClusterGuard) GetClusterName() *string {
	return f.Spec.ClusterGuardControllerConfig.ClusterName
}

// GetConditions returns a pointer to the Conditions slice for status updates
func (f *FalconClusterGuard) GetConditions() *[]metav1.Condition {
	return &f.Status.Conditions
}

// --- Node sensor types ---

// FalconClusterGuardUpdateStrategy defines the update strategy for the node sensor DaemonSet.
type FalconClusterGuardUpdateStrategy struct {
	// +kubebuilder:default=RollingUpdate
	// +kubebuilder:validation:Enum=RollingUpdate;OnDelete
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Type appsv1.DaemonSetUpdateStrategyType `json:"type,omitempty"`
	// +kubebuilder:default={"maxUnavailable": 1, "maxSurge": 0}
	RollingUpdate appsv1.RollingUpdateDaemonSet `json:"rollingUpdate,omitempty"`
}

// FalconClusterGuardNodeServiceAccount defines service account metadata for the node sensor.
type FalconClusterGuardNodeServiceAccount struct {
	// Define annotations that will be passed down to the Service Account. This is useful for passing along AWS IAM Role or GCP Workload Identity.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Annotations map[string]string `json:"annotations,omitempty"`
}

// FalconClusterGuardResourceList defines CPU, memory, and ephemeral-storage resource quantities.
type FalconClusterGuardResourceList struct {
	// Minimum allowed is 250m.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Pattern="^(([0-9]{4,}|[2-9][5-9][0-9])m$)|[0-9]+$"
	CPU string `json:"cpu,omitempty"`

	// Minimum allowed is 500Mi.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Pattern="^(([5-9][0-9]{2}[Mi]+)|([0-9.]+[iEGTP]+))|(([5-9][0-9]{8})|([0-9]{10,}))$"
	Memory string `json:"memory,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	EphemeralStorage string `json:"ephemeral-storage,omitempty"`
}

// FalconClusterGuardResources defines resource requests and limits for the node sensor.
type FalconClusterGuardResources struct {
	// Sets the resource limits for the DaemonSet Sensor. Only applies when using the eBPF backend.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Limits FalconClusterGuardResourceList `json:"limits,omitempty"`

	// Sets the resource requests for the DaemonSet Sensor. Only applies when using the eBPF backend.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Requests FalconClusterGuardResourceList `json:"requests,omitempty"`
}

// IsConfigured returns true if any resource value has been set.
func (r FalconClusterGuardResources) IsConfigured() bool {
	return r.Limits != (FalconClusterGuardResourceList{}) || r.Requests != (FalconClusterGuardResourceList{})
}

// FalconClusterGuardAutoPilot defines GKE Autopilot settings for the node sensor.
type FalconClusterGuardAutoPilot struct {
	// Enables the use of GKE Autopilot.
	// +kubebuilder:default=false
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Enabled *bool `json:"autopilot,omitempty"`

	// Version of the GKE AutoPilot DaemonSet for allow list troubleshooting purposes.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Pattern="^v[0-9]+\\.[0-9]+\\.[0-9]+$"
	DeployAllowListVersion *string `json:"deployAllowListVersion,omitempty"`

	// Version of the GKE AutoPilot Cleanup DaemonSet for allow list troubleshooting purposes.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Pattern="^v[0-9]+\\.[0-9]+\\.[0-9]+$"
	CleanupAllowListVersion *string `json:"cleanupAllowListVersion,omitempty"`
}

// FalconClusterGuardPriorityClassConfig defines priority class settings for the node sensor DaemonSet.
type FalconClusterGuardPriorityClassConfig struct {
	// Enables the operator to deploy a PriorityClass instead of rolling your own. Default is false.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Deploy Priority Class to cluster",order=2
	Deploy *bool `json:"deploy,omitempty"`

	// Name of the priority class to use for the DaemonSet.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Name of the Priority Class to use",order=2
	Name string `json:"name,omitempty"`

	// Value of the priority class to use for the DaemonSet. Requires the Deploy field to be set to true.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Priority Class Value",order=3
	Value *int32 `json:"value,omitempty"`
}

// --- Admission controller types ---

// FalconClusterGuardControllerServiceAccount defines service account metadata for the admission controller.
type FalconClusterGuardControllerServiceAccount struct {
	// Define annotations that will be passed down to the Service Account. This is useful for passing along AWS IAM Role or GCP Workload Identity.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Service Account Annotations",order=1
	Annotations map[string]string `json:"annotations,omitempty"`
}

// FalconClusterGuardControllerRollingUpdate defines rolling update parameters for the admission controller Deployment.
type FalconClusterGuardControllerRollingUpdate struct {
	// The maximum number of pods that can be unavailable during the update.
	// +kubebuilder:default:=0
	// +optional
	MaxUnavailable *intstr.IntOrString `json:"maxUnavailable,omitempty"`

	// The maximum number of pods that can be scheduled above the desired number of pods.
	// +kubebuilder:default:=1
	// +optional
	MaxSurge *intstr.IntOrString `json:"maxSurge,omitempty"`
}

// FalconClusterGuardControllerUpdateStrategy defines the update strategy for the admission controller Deployment.
type FalconClusterGuardControllerUpdateStrategy struct {
	// RollingUpdate is used to specify the strategy used to roll out a deployment.
	// +kubebuilder:default:={}
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Admission Controller deployment update configuration",order=1,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:updateStrategy"}
	RollingUpdate FalconClusterGuardControllerRollingUpdate `json:"rollingUpdate,omitempty"`
}

// FalconClusterGuardControllerTLS defines TLS settings for the admission controller.
type FalconClusterGuardControllerTLS struct {
	// Validity of the TLS certificate in days. Default is 3650 days.
	// +kubebuilder:validation:XIntOrString
	// +kubebuilder:validation:Pattern="^[0-9]{1,4}$"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Admission Controller TLS Validity Length (days)",order=1,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number"}
	Validity *int `json:"validity,omitempty"`
}

// FalconClusterGuardControllerNamespace defines the namespace ignore list for the admission controller.
type FalconClusterGuardControllerNamespace struct {
	// Configure a list of namespaces to ignore admission control.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Ignore Namespace List",order=1
	Namespaces []string `json:"namespaces,omitempty"`
}
