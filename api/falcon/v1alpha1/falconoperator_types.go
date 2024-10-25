package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DeployAdmissionControllerDefault = true
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// FalconOperatorSpec defines the desired state of FalconOperator
// +k8s:openapi-gen=true
type FalconOperatorSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// FalconAPI configures connection from your local Falcon operator to CrowdStrike Falcon platform.
	//
	// When configured, it will pull the sensor from registry.crowdstrike.com and deploy the appropriate sensor to the cluster.
	//
	// If using the API is not desired, the sensor can be manually configured.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Platform API Configuration",order=2
	FalconAPI *FalconAPI `json:"falcon_api,omitempty"`

	// Registry configures container image registry to which the Admission Controller image will be pushed.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Admission Controller Registry Configuration",order=3
	Registry RegistrySpec `json:"registry,omitempty"`

	// Determines if Falcon Admission Controller is deployed
	// +kubebuilder:default:=true
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Deploy Falcon Admission Controller",order=4
	DeployAdmissionController *bool `json:"deployAdmissionController,omitempty"`

	// Falcon Admission Controller Configuration
	// +kubebuilder:default={"registry":{"type": "crowdstrike"}}
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Admission Controller Configuration",order=5
	FalconAdmissionConfig *FalconAdmissionSpec `json:"falconAdmissionConfig,omitempty"`
}

// FalconOperatorStatus defines the observed state of FalconOperator
type FalconOperatorStatus struct {
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

// FalconOperator is the Schema for the falconoperators API
type FalconOperator struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FalconOperatorSpec   `json:"spec,omitempty"`
	Status FalconOperatorStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// FalconOperatorList contains a list of FalconOperator
type FalconOperatorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FalconOperator `json:"items"`
}

func (security FalconOperator) GetDeployAdmissionController() bool {
	if security.Spec.DeployAdmissionController == nil {
		return DeployAdmissionControllerDefault
	}

	return *security.Spec.DeployAdmissionController
}

func init() {
	SchemeBuilder.Register(&FalconOperator{}, &FalconOperatorList{})
}
