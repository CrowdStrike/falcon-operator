package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.
type FalconContainerInjectorSpec struct {
	// Define annotations that will be passed down to injector service account. This is useful for passing along AWS IAM Role or GCP Workload Identity.
	SAAnnotations map[string]string `json:"sa_annotations,omitempty"`
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
	// Injector represents additional configuration for Falcon Container Injector
	Injector *FalconContainerInjectorSpec `json:"injector,omitempty"`

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
	// PhaseValidating represents the state when falcon-operator validates injector pod installation
	PhaseValidating FalconContainerStatusPhase = "VALIDATING"
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
