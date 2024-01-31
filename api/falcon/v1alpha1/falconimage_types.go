package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// FalconImageSpec defines the desired state of FalconImage
type FalconImageSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of FalconImage. Edit falconimage_types.go to remove/update
	Foo string `json:"foo,omitempty"`
}

// FalconImageStatus defines the observed state of FalconImage
type FalconImageStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// FalconImage is the Schema for the falconimages API
type FalconImage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FalconImageSpec   `json:"spec,omitempty"`
	Status FalconImageStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// FalconImageList contains a list of FalconImage
type FalconImageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FalconImage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FalconImage{}, &FalconImageList{})
}
