package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// FalconDeploymentSpec defines the desired state of FalconDeployment
// +k8s:openapi-gen=true
type FalconDeploymentSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// FalconAPI configures connection from your local Falcon operator to CrowdStrike Falcon platform.
	//
	// When configured, it will pull the sensor from registry.crowdstrike.com and deploy the appropriate sensor to the cluster.
	//
	// If using the API is not desired, the sensor can be manually configured.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Platform API Configuration",order=1
	FalconAPI *FalconAPI `json:"falcon_api,omitempty"`

	// Registry configures container image registry to which registry image will be pushed.
	// +kubebuilder:default:={"type": "crowdstrike"}
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Registry Configuration for FalconAdmission, FalconImageanalyzer, and FalconContainer",order=2
	Registry RegistrySpec `json:"registry,omitempty"`

	// FalconSecret config is used to inject k8s secrets with sensitive data for the FalconSensor and the FalconAPI.
	// The following Falcon values are supported by k8s secret injection:
	//   falcon-cid
	//   falcon-provisioning-token
	//   falcon-client-id
	//   falcon-client-secret
	// +kubebuilder:default={"enabled": false}
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Platform Secrets Configuration",order=3
	FalconSecret FalconSecret `json:"falconSecret,omitempty"`

	// Determines if Falcon Admission Controller is deployed
	// +kubebuilder:default:=true
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Deploy Falcon Admission Controller",order=4
	DeployAdmissionController *bool `json:"deployAdmissionController,omitempty"`

	// Determines if Falcon Node Sensor is deployed
	// +kubebuilder:default:=true
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Deploy Falcon Node Sensor",order=5
	DeployNodeSensor *bool `json:"deployNodeSensor,omitempty"`

	// Determines if Falcon Node Sensor is deployed
	// +kubebuilder:default:=true
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Deploy Falcon Image Analyzer",order=6
	DeployImageAnalyzer *bool `json:"deployImageAnalyzer,omitempty"`

	// Determines if Falcon Container Sensor is deployed
	// +kubebuilder:default:=false
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Deploy Falcon Container Sensor",order=7
	DeployContainerSensor *bool `json:"deployContainerSensor,omitempty"`

	// Falcon Admission Controller Configuration
	// +kubebuilder:default:={}
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Admission Controller Configuration",order=8
	FalconAdmission FalconAdmissionSpec `json:"falconAdmission,omitempty"`

	// Falcon Node Sensor Controller Configuration
	// +kubebuilder:default:={}
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Node Sensor Configuration",order=9
	FalconNodeSensor FalconNodeSensorSpec `json:"falconNodeSensor,omitempty"`

	// Falcon Image Analyzer Configuration
	// +kubebuilder:default:={}
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Image Analyzer Configuration",order=10
	FalconImageAnalyzer FalconImageAnalyzerSpec `json:"falconImageAnalyzer,omitempty"`

	// Falcon Container Sensor Configuration
	// +kubebuilder:default:={}
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Container Sensor Configuration",order=11
	FalconContainerSensor FalconContainerSpec `json:"falconContainerSensor,omitempty"`
}

// FalconDeploymentStatus defines the observed state of FalconDeployment
type FalconDeploymentStatus struct {
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

// FalconDeployment is the Schema for the falcondeployments API
type FalconDeployment struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FalconDeploymentSpec   `json:"spec,omitempty"`
	Status FalconDeploymentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// FalconDeploymentList contains a list of FalconDeployment
type FalconDeploymentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FalconDeployment `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FalconDeployment{}, &FalconDeploymentList{})
}
