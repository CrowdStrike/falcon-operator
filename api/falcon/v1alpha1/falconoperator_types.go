package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	DeployAdmissionControllerDefault = true
	DeployNodeSensorDefault          = true
	DeployImageAnalyzerDefault       = true
	DeployContainerSensorDefault     = true
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

	// Registry configures container image registry to which registry image will be pushed.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Registry Configuration for FalconAdmission, FalconImageanalyzer, and FalconContainer",order=3
	Registry RegistrySpec `json:"registry,omitempty"`

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
	// +kubebuilder:default:=true
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Deploy Falcon Container Sensor",order=7
	DeployContainerSensor *bool `json:"deployContainerSensor,omitempty"`

	// Falcon Admission Controller Configuration
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Admission Controller Configuration",order=8
	FalconAdmission FalconAdmissionSpec `json:"falconAdmission,omitempty"`

	// Falcon Admission Controller Configuration
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Node Sensor Configuration",order=9
	FalconNodeSensor FalconNodeSensorSpec `json:"falconNodeSensor,omitempty"`

	// Falcon Image Analyzer Configuration
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Image Analyzer Configuration",order=10
	FalconImageAnalyzer FalconImageAnalyzerSpec `json:"falconImageAnalyzer,omitempty"`

	// Falcon Container Sensor Configuration
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Container Sensor Configuration",order=11
	FalconContainerSensor FalconContainerSpec `json:"falconContainerSensor,omitempty"`
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

func (security *FalconOperator) DeployAdmissionController() bool {
	if security.Spec.DeployAdmissionController == nil {
		return DeployAdmissionControllerDefault
	}

	return *security.Spec.DeployAdmissionController
}

func (security *FalconOperator) DeployNodeSensor() bool {
	if security.Spec.DeployNodeSensor == nil {
		return DeployNodeSensorDefault
	}

	return *security.Spec.DeployNodeSensor
}

func (security *FalconOperator) DeployImageAnalyzer() bool {
	if security.Spec.DeployImageAnalyzer == nil {
		return DeployImageAnalyzerDefault
	}

	return *security.Spec.DeployImageAnalyzer
}

func (security *FalconOperator) DeployContainerSensor() bool {
	if security.Spec.DeployContainerSensor == nil {
		return DeployContainerSensorDefault
	}

	return *security.Spec.DeployContainerSensor
}

func init() {
	SchemeBuilder.Register(&FalconOperator{}, &FalconOperatorList{})
}

func NewFalconOperatorSpec() FalconOperatorSpec {
	return FalconOperatorSpec{
		Registry: RegistrySpec{
			Type: "crowdstrike",
		},
		FalconAdmission:       NewFalconAdmissionSpec(),
		FalconNodeSensor:      NewFalconNodeSensorSpec(),
		FalconImageAnalyzer:   NewImageAnalyzerSpec(),
		FalconContainerSensor: NewFalconContainerSpec(),
	}
}
