package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// FalconNodeSensorSpec defines the desired state of FalconNodeSensor
type FalconNodeSensorSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of FalconNodeSensor. Edit falconnodesensor_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// FalconNodeSensorStatus defines the observed state of FalconNodeSensor
type FalconNodeSensorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// FalconNodeSensor is the Schema for the falconnodesensors API
type FalconNodeSensor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FalconNodeSensorSpec   `json:"spec,omitempty"`
	Status FalconNodeSensorStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// FalconNodeSensorList contains a list of FalconNodeSensor
type FalconNodeSensorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FalconNodeSensor `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FalconNodeSensor{}, &FalconNodeSensorList{})
}
