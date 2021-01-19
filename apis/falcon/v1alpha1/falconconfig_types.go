/*
Copyright 2021 CrowdStrike
*/

package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// FalconConfigSpec defines the desired state of FalconConfig
type FalconConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of FalconConfig. Edit FalconConfig_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// FalconConfigStatus defines the observed state of FalconConfig
type FalconConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
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
