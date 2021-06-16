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
	// Falcon Customer ID (CID)
	// +kubebuilder:validation:Pattern="^[0-9a-fA-F]{32}-[0-9a-fA-F]{2}$"
	CID string `json:"cid"`
}

// RegistryTLSSpec configures TLS for registry pushing
type RegistryTLSSpec struct {
	// Allow pushing to docker registries over HTTPS with failed TLS verification. Note that this does not affect other TLS connections.
	InsecureSkipVerify bool `json:"insecure_skip_verify,omitempty"`
}

// RegistrySpec configures container image registry to which the Falcon Container image will be pushed
type RegistrySpec struct {
	// TLS configures TLS connection for push of Falcon Container image to the registry
	TLS RegistryTLSSpec `json:"tls,omitempty"`
}

// FalconContainerSpec defines the desired state of FalconContainer
type FalconContainerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// FalconAPI configures connection from your local Falcon operator to CrowdStrike Falcon platform.
	FalconAPI FalconAPI `json:"falcon_api"`
	// Registry configures container image registry to which the Falcon Container image will be pushed
	Registry RegistrySpec `json:"registry,omitempty"`
}

// Represents the status of Falcon deployment
type FalconContainerStatusPhase string

const (
	// PhasePending represents the deployment to be started
	PhasePending FalconContainerStatusPhase = "PENDING"
	// PhaseBuilding represents the deployment before the falcon image is successfully fetched
	PhaseBuilding FalconContainerStatusPhase = "BUILDING"
	// PhaseConfiguring represents the state when injector/installer is being run
	PhaseConfiguring = "CONFIGURING"
	// PhaseDeploying represents the state when injector is being deployed on the cluster
	PhaseDeploying = "DEPLOYING"
	// PhaseDone represents the Falcon Protection being successfully installed
	PhaseDone FalconContainerStatusPhase = "DONE"
)

// FalconContainerStatus defines the observed state of FalconContainer
type FalconContainerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Phase or the status of the deployment
	Phase FalconContainerStatusPhase `json:"phase,omitempty"`

	// ErrorMessage informs user of the last notable error. Users are welcomed to see the operator logs
	// to understand the full context.
	ErrorMessage string `json:"errormsg,omitempty"`
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

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
