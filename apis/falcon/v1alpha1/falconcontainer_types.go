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

type RegistryTypeSpec string

const (
	// RegistryTypeOpenshift represents OpenShift Image Stream
	RegistryTypeOpenshift RegistryTypeSpec = "openshift"
	// RegistryTypeGCR represents Google Container Registry
	RegistryTypeGCR RegistryTypeSpec = "gcr"
	// RegistryTypeECR represents AWS Elastic Container Registry
	RegistryTypeECR RegistryTypeSpec = "ecr"
)

// RegistrySpec configures container image registry to which the Falcon Container image will be pushed
type RegistrySpec struct {
	// Type of the registry to be used
	// +kubebuilder:validation:Enum=ecr;gcr;openshift
	Type RegistryTypeSpec `json:"type"`

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

	// InstallerArgs are passed directly down to the Falcon Container Installer. Users are advised to consult Falcon Container documentation to learn about available command line arguments at https://falcon.crowdstrike.com/documentation/146/falcon-container-sensor-for-linux
	InstallerArgs []string `json:"installer_args,omitempty"`
	// Falcon Container Version. The latest version will be selected when version specifier is missing.
	Version *string `json:"version,omitempty"`
}

// Represents the status of Falcon deployment
type FalconContainerStatusPhase string

const (
	// PhasePending represents the deployment to be started
	PhasePending FalconContainerStatusPhase = "PENDING"
	// PhaseBuilding represents the deployment before the falcon image is successfully fetched
	PhaseBuilding FalconContainerStatusPhase = "BUILDING"
	// PhaseConfiguring represents the state when injector/installer is being run
	PhaseConfiguring FalconContainerStatusPhase = "CONFIGURING"
	// PhaseDeploying represents the state when injector is being deployed on the cluster
	PhaseDeploying FalconContainerStatusPhase = "DEPLOYING"
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

	Version *string `json:"version,omitempty"`

	// RetryAttempt is number of previous failed attempts. Valid values: 0-5
	RetryAttempt *uint8 `json:"retry_attempt,omitempty"`
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
