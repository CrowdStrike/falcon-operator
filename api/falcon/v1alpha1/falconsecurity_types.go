package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DeployAdmissionControllerDefault = true
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// FalconSecuritySpec defines the desired state of FalconSecurity
type FalconSecuritySpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// FalconAPI configures connection from your local Falcon operator to CrowdStrike Falcon platform.
	//
	// When configured, it will pull the sensor from registry.crowdstrike.com and deploy the appropriate sensor to the cluster.
	//
	// If using the API is not desired, the sensor can be manually configured.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Platform API Configuration",order=1
	FalconAPI *FalconAPI `json:"falcon_api,omitempty"`

	// Determines if Falcon Admission Controller is deployed
	// +kubebuilder:default:=true
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Deploy Falcon Admission Controller",order=2
	DeployAdmissionController *bool `json:"deployAdmissionController,omitempty"`
}

// FalconSecurityStatus defines the observed state of FalconSecurity
type FalconSecurityStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// FalconSecurity is the Schema for the falconsecurities API
type FalconSecurity struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FalconSecuritySpec   `json:"spec,omitempty"`
	Status FalconSecurityStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// FalconSecurityList contains a list of FalconSecurity
type FalconSecurityList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FalconSecurity `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FalconSecurity{}, &FalconSecurityList{})
}

func (security FalconSecuritySpec) GetDeployAdmissionController() bool {
	if security.DeployAdmissionController == nil {
		return DeployAdmissionControllerDefault
	}

	return *security.DeployAdmissionController
}
