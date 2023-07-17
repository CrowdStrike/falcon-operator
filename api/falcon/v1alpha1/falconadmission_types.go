package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// FalconAdmissionSpec defines the desired state of FalconAdmission
type FalconAdmissionSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of FalconAdmission. Edit falconadmission_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// FalconAdmissionStatus defines the observed state of FalconAdmission
type FalconAdmissionStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// FalconAdmission is the Schema for the falconadmissions API
type FalconAdmission struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FalconAdmissionSpec   `json:"spec,omitempty"`
	Status FalconAdmissionStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// FalconAdmissionList contains a list of FalconAdmission
type FalconAdmissionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FalconAdmission `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FalconAdmission{}, &FalconAdmissionList{})
}
