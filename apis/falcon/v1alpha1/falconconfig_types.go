package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// FalconAPI configures connection from your local Falcon operator to CrowdStrike Falcon platform.
type FalconAPI struct {
	// CloudRegion defines CrowdStrike Falcon Cloud Region to which the operator will connect to
	// +kubebuilder:validation:Enum=us-1;us-2;eu-1;us-gov-1
	CloudRegion string `json:"cloud_region"`
	// Falcon OAuth2 API Client ID
	ClientId string `json:"client_id"`
	// Falcon OAuth2 API Client Secret
	ClientSecret string `json:"client_secret"`
}

// LinuxContainerSpec configures Falcon Container Sensor product installation on your cluster
type LinuxContainerSpec struct {
	Enabled bool `json:"enabled"`
	// Container Registry to which falcon-operator will push falcon-container sensor and from which falcon-container sensor will be consumed by pods
	Registry string `json:"registry"`
}

type ContainerItem struct {
	Image   string      `json:"image"`
	Created metav1.Time `json:"created"`
	Tag     string      `json:"tag"`
}

type RegistryStatus struct {
	Location string          `json:"location"`
	Items    []ContainerItem `json:"items"`
}

// LinuxContainerStatus defines observed state of FalconConfig
type LinuxContainerStatus struct {
	Registry *RegistryStatus `json:"registry"`
}

// WorkloadProtectionSpec configures workload protection on the cluster
type WorkloadProtectionSpec struct {
	// LinuxContainerSpec configures Falcon Container Sensor product installation on your cluster
	LinuxContainerSpec *LinuxContainerSpec `json:"linux_container,omitempty"`
}

// WorkloadProtectionStatus defines observed state of workload protection on the cluster
type WorkloadProtectionStatus struct {
	LinuxContainerStatus *LinuxContainerStatus `json:"linux_container,omitempty"`
}

// FalconConfigSpec defines the desired state of FalconConfig
type FalconConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// FalconAPI configures connection from your local Falcon operator to CrowdStrike Falcon platform.
	FalconAPI FalconAPI `json:"falcon_api"`
	// WorkloadProtectionSpec configures workload protection on the cluster
	WorkloadProtectionSpec WorkloadProtectionSpec `json:"workload_protection"`
}

// Represents the status of Falcon deployment
type FalconConfigStatusPhase string

const (
	// PhasePending represents the deployment to be started
	PhasePending FalconConfigStatusPhase = "PENDING"
	// PhaseBuilding represents the deployment before the falcon image is successfully fetched
	PhaseBuilding FalconConfigStatusPhase = "BUILDING"
	// PhaseDone represents the Falcon Protection being successfully installed
	PhaseDone FalconConfigStatusPhase = "DONE"
)

// FalconConfigStatus defines the observed state of FalconConfig
type FalconConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Phase or the status of the deployment
	Phase FalconConfigStatusPhase `json:"phase,omitempty"`

	// ErrorMessage informs user of the last notable error. Users are welcomed to see the operator logs
	// to understand the full context.
	ErrorMessage string `json:"errormsg,omitempty"`

	WorkloadProtectionStatus *WorkloadProtectionStatus `json:"workload_protection,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// FalconConfig is the Schema for the falconconfigs API
type FalconConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FalconConfigSpec   `json:"spec,omitempty"`
	Status FalconConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// FalconConfigList contains a list of FalconConfig
type FalconConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FalconConfig `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FalconConfig{}, &FalconConfigList{})
}
