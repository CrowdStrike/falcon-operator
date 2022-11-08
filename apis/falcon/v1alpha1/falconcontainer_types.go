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

	// FalconAPI configures connection from your local Falcon operator to CrowdStrike Falcon platform.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Platform API Configuration",order=1
	FalconAPI FalconAPI `json:"falcon_api"`

	// Registry configures container image registry to which the Falcon Container image will be pushed
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Container Image Registry Configuration",order=2
	Registry RegistrySpec `json:"registry,omitempty"`

	// Injector represents additional configuration for Falcon Container Injector
	// +kubebuilder:default:={imagePullPolicy:Always}
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Container Injector Configuration",order=3
	Injector FalconContainerInjectorSpec `json:"injector,omitempty"`

	// Falcon Container Version Pinning. If not set to false, once a sensor version is set, it is used until manually adjusted.
	// +kubebuilder:default:=true
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Container Image Version Pinning",order=4
	VersionPinning bool `json:"versionPinning,omitempty"`

	// +kubebuilder:validation:Pattern="^.*:.*$"
	// +operator-sdk:cv:customresourcedefinitions:type=spec,displayName="Falcon Container Image URI",order=5
	Image *string `json:"image,omitempty"`

	// Falcon Container Version. The latest version will be selected when version specifier is missing; ignored when Image is set.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Container Image Version",order=6
	Version *string `json:"version,omitempty"`
}

type FalconContainerInjectorSpec struct {
	// Define annotations that will be passed down to injector service account. This is useful for passing along AWS IAM Role or GCP Workload Identity.
	// +kubebuilder:default:={name:default}
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

	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falconctl Opts options",order=9
	FalconctlOpts string `json:"falconctlOpts,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Container Additional Environment Variables",order=10
	AdditionalEnvironmentVariables *map[string]string `json:"additionalEnvironmentVariables,omitempty"`

	// +kubebuilder:default=false
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Disable Default Namespace Injection",order=11
	DisableDefaultNSInjection bool `json:"disableDefaultNamespaceInjection,omitempty"`

	// +kubebuilder:default=false
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Disable Default Pod Injection",order=12
	DisableDefaultPodInjection bool `json:"disableDefaultPodInjection,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Azure Config file path",order=13
	AzureConfigPath string `json:"azureConfigPath,omitempty"`
}

type FalconContainerServiceAccount struct {
	// Define annotations that will be passed down to the Service Account. This is useful for passing along AWS IAM Role or GCP Workload Identity.
	// +kubebuilder:default=default
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Service Account Name",order=1
	Name string `json:"name,omitempty"`
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Annotations map[string]string `json:"annotations,omitempty"`
}

type FalconContainerInjectorTLS struct {
	// +kubebuilder:validation:XIntOrString
	// +kubebuilder:validation:Pattern="^[0-9]{1-4}$"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Container Injector TLS Validity Length (days)",order=1
	Validity *int `json:"validity,omitempty"`
}

const (
	PhaseReconciling FalconContainerStatusPhase = "RECONCILING"
	PhaseError       FalconContainerStatusPhase = "ERROR"
	PhaseDone        FalconContainerStatusPhase = "DONE"
)

// Represents the status of Falcon deployment
type FalconContainerStatusPhase string

// FalconContainerStatus defines the observed state of FalconContainer
type FalconContainerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Phase or the status of the deployment
	Phase FalconContainerStatusPhase `json:"phase,omitempty"`

	// ErrorMessage informs user of the last notable error. Users are welcomed to see the operator logs
	// to understand the full context.
	ErrorMessage string `json:"errormsg,omitempty"`

	Version *string `json:"version,omitempty"`

	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase",description="Phase of deployment"
//+kubebuilder:printcolumn:name="Version",type="string",JSONPath=".status.version",description="Version of Falcon Container"
//+kubebuilder:printcolumn:name="Error",type="string",JSONPath=".status.errormsg",description="Last error message"

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
