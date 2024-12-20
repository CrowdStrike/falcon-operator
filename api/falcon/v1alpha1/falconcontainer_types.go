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

	// Registry configures container image registry to which the Falcon Container image will be pushed
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Container Image Registry Configuration",order=3
	Registry RegistrySpec `json:"registry,omitempty"`

	// Injector represents additional configuration for Falcon Container Injector
	// +kubebuilder:default:={}
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Container Injector Configuration",order=4
	Injector FalconContainerInjectorSpec `json:"injector,omitempty"`

	// +kubebuilder:validation:Pattern="^.*:.*$"
	// +operator-sdk:cv:customresourcedefinitions:type=spec,displayName="Falcon Container Image URI",order=5
	Image *string `json:"image,omitempty"`

	// Falcon Container Version. The latest version will be selected when version specifier is missing; ignored when Image is set.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Container Image Version",order=6
	Version *string `json:"version,omitempty"`

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
