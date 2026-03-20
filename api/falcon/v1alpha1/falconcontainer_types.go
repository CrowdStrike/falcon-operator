package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// FalconContainerSpec defines the desired state of FalconContainer
// +k8s:openapi-gen=true
type FalconContainerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Namespace where the Falcon Sensor should be installed.
	// For best security practices, this should be a dedicated namespace that is not used for any other purpose.
	// It also should not be the same namespace where the Falcon Operator, or other Falcon resources are deployed.
	// +kubebuilder:default:=falcon-system
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=1,xDescriptors={"urn:alm:descriptor:io.kubernetes:Namespace"}
	InstallNamespace string `json:"installNamespace,omitempty"`

	// +kubebuilder:default:={}
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Sensor Configuration",order=1
	Falcon FalconSensor `json:"falcon,omitempty"`

	// FalconAPI configures connection from your local Falcon operator to CrowdStrike Falcon platform.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Platform API Configuration",order=2
	FalconAPI *FalconAPI `json:"falcon_api,omitempty"`

	// FalconSecret config is used to inject k8s secrets with sensitive data for the FalconSensor and the FalconAPI.
	// The following Falcon values are supported by k8s secret injection:
	//   falcon-cid
	//   falcon-provisioning-token
	//   falcon-client-id
	//   falcon-client-secret
	// +kubebuilder:default={"enabled": false}
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Platform Secrets Configuration",order=5
	FalconSecret FalconSecret `json:"falconSecret,omitempty"`

	// Registry configures container image registry to which the Falcon Container image will be pushed
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Container Image Registry Configuration",order=3
	Registry RegistrySpec `json:"registry,omitempty"`

	// Injector represents additional configuration for Falcon Container Injector
	// +kubebuilder:default:={}
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Container Injector Configuration",order=4
	Injector FalconContainerInjectorSpec `json:"injector,omitempty"`

	// +kubebuilder:validation:Pattern="^.*:.*$"
	// +operator-sdk:cv:customresourcedefinitions:type=spec,displayName="Falcon Container Image URI",order=6
	Image *string `json:"image,omitempty"`

	// Falcon Container Version. The latest version will be selected when version specifier is missing; ignored when Image is set.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Container Image Version",order=7
	Version *string `json:"version,omitempty"`

	// Specifies node affinity for scheduling the Container Sensor. Only amd64 linux nodes are supported.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=7
	NodeAffinity *corev1.NodeAffinity `json:"nodeAffinity,omitempty"`

	// Advanced configures various options that go against industry practices or are otherwise not recommended for use.
	// Adjusting these settings may result in incorrect or undesirable behavior. Proceed at your own risk.
	// For more information, please see https://github.com/CrowdStrike/falcon-operator/blob/main/docs/ADVANCED.md.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Container Advanced Settings"
	Advanced FalconAdvanced `json:"advanced,omitempty"`
}

type FalconContainerInjectorSpec struct {
	// Define annotations that will be passed down to injector service account. This is useful for passing along AWS IAM Role or GCP Workload Identity.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Service Account Configuration",order=1
	ServiceAccount FalconContainerServiceAccount `json:"serviceAccount,omitempty"`

	// +kubebuilder:default:=4433
	// +kubebuilder:validation:XIntOrString
	// +kubebuilder:validation:Minimum:=0
	// +kubebuilder:validation:Maximum:=65535
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Container Injector Listen Port",order=2
	ListenPort *int32 `json:"listenPort,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Container Injector TLS Configuration",order=3
	TLS FalconContainerInjectorTLS `json:"tls,omitempty"`

	// +kubebuilder:default:=Always
	// +kubebuilder:validation:Enum=Always;IfNotPresent;Never
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Container Image Pull Policy",order=4
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// +kubebuilder:default=crowdstrike-falcon-pull-secret
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Container Image Pull Secret Name",order=5
	ImagePullSecretName string `json:"imagePullSecret,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Container Shared Log Volume",order=6
	LogVolume *corev1.Volume `json:"logVolume,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Container Injector Resources",order=7
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Container Sensor Resources",order=8
	SensorResources *corev1.ResourceRequirements `json:"sensorResources,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Container Additional Environment Variables",order=9
	AdditionalEnvironmentVariables *map[string]string `json:"additionalEnvironmentVariables,omitempty"`

	// +kubebuilder:default=false
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Disable Default Namespace Injection",order=10
	DisableDefaultNSInjection bool `json:"disableDefaultNamespaceInjection,omitempty"`

	// +kubebuilder:default=false
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Disable Default Pod Injection",order=11
	DisableDefaultPodInjection bool `json:"disableDefaultPodInjection,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Azure Config file path",order=12
	AzureConfigPath string `json:"azureConfigPath,omitempty"`

	// +kubebuilder:default:=2
	// +kubebuilder:validation:XIntOrString
	// +kubebuilder:validation:Minimum:=0
	// +kubebuilder:validation:Maximum:=65535
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Injector replica count",order=13
	Replicas *int32 `json:"replicas,omitempty"`

	// +kubebuilder:default=false
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Enable Alternate Mount Path", order=14
	AlternateMountPath bool `json:"alternateMountPath,omitempty"`

	// AITap configures AI Detection and Response (AI-DR) functionality
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="AITap AI-DR Configuration", order=15
	AITap AITapSpec `json:"aitap,omitempty"`

	// GKE Autopilot mode. When enabled, uses fixed secret name "falcon-node-sensor-aitap-aidr-secret"
	// required by GKE Autopilot WorkloadAllowlists
	// +kubebuilder:default=false
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Enable GKE Autopilot Mode",order=6,xDescriptors="urn:alm:descriptor:com.tectonic.ui:booleanSwitch"
	GKEAutopilot bool `json:"gkeAutopilot,omitempty"`
}

type FalconContainerServiceAccount struct {
	// Define annotations that will be passed down to the Service Account. This is useful for passing along AWS IAM Role or GCP Workload Identity.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Annotations map[string]string `json:"annotations,omitempty"`
}

type FalconContainerInjectorTLS struct {
	// +kubebuilder:validation:XIntOrString
	// +kubebuilder:validation:Pattern="^[0-9]{1-4}$"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Container Injector TLS Validity Length (days)",order=1
	Validity *int `json:"validity,omitempty"`
}

// AITapSpec defines the AI Detection and Response configuration
// +k8s:openapi-gen=true
type AITapSpec struct {
	// AI-DR Collector API token for the Application Collector
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="AI-DR Collector API Token",order=1
	AidrCollectorApiToken string `json:"aidrCollectorApiToken,omitempty"`

	// AI-DR Collector Base API URL for the Application Collector (optional)
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="AI-DR Collector Base API URL",order=2
	AidrCollectorBaseApiUrl string `json:"aidrCollectorBaseApiUrl,omitempty"`

	// Configure the list of namespaces that should have access to the AI-DR credentials.
	// This is a comma-separated list. For example: "ns1,ns2,ns3"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="AITap Namespaces (comma-separated)",order=3
	Namespaces string `json:"namespaces,omitempty"`

	// Create the AI-DR secret in all Namespaces instead of using specific namespaces list
	// +kubebuilder:default=false
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Enable AITap in All Namespaces",order=4,xDescriptors="urn:alm:descriptor:com.tectonic.ui:booleanSwitch"
	AllNamespaces bool `json:"allNamespaces,omitempty"`

	// AI-DR Kubernetes secret name override. If not specified, the secret name will be automatically determined
	// based on GKE Autopilot detection or release name
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Custom AITap AI-DR Secret Name",order=5
	AidrSecretName string `json:"aidrSecretName,omitempty"`

	// Use an externally managed AI-DR secret instead of having the operator create and manage it.
	// When true, the operator assumes a secret with the configured name already exists in target
	// namespaces and will not create or manage secrets.
	// +kubebuilder:default=false
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Use External Secret",order=6,xDescriptors="urn:alm:descriptor:com.tectonic.ui:booleanSwitch"
	UseExternalSecret bool `json:"useExternalSecret,omitempty"`
}

// FalconContainerStatus defines the observed state of FalconContainer
// +k8s:openapi-gen=true
type FalconContainerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Version of the CrowdStrike Falcon Sensor
	Sensor *string `json:"sensor,omitempty"`

	// Version of the CrowdStrike Falcon Operator
	Version string `json:"version,omitempty"`

	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster
//+kubebuilder:printcolumn:name="Operator Version",type="string",JSONPath=".status.version",description="Version of the Operator"
//+kubebuilder:printcolumn:name="Falcon Sensor",type="string",JSONPath=".status.sensor",description="Version of the Falcon Container"

// FalconContainer is the Schema for the falconcontainers API
type FalconContainer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FalconContainerSpec   `json:"spec,omitempty"`
	Status FalconContainerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// FalconContainerList contains a list of FalconContainer
type FalconContainerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FalconContainer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FalconContainer{}, &FalconContainerList{})
}

func (fc *FalconContainer) GetFalconSecretSpec() FalconSecret {
	return fc.Spec.FalconSecret
}

func (fc *FalconContainer) GetFalconAPISpec() *FalconAPI {
	return fc.Spec.FalconAPI
}

func (fc *FalconContainer) SetFalconAPISpec(falconApiSpec *FalconAPI) {
	fc.Spec.FalconAPI = falconApiSpec
}

func (fc *FalconContainer) GetFalconSpec() FalconSensor {
	return fc.Spec.Falcon
}

func (fc *FalconContainer) SetFalconSpec(falconSpec FalconSensor) {
	fc.Spec.Falcon = falconSpec
}
