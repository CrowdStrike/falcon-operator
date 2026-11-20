package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InstallGuardOwnerSpec is empty. InstallGuardOwner holds no data.
type InstallGuardOwnerSpec struct{}

// InstallGuardOwnerStatus is empty. InstallGuardOwner holds no data.
type InstallGuardOwnerStatus struct{}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster

// InstallGuardOwner is the owner of the FCG install guard ValidatingWebhookConfiguration created by the
// clusterguard controller when uninstall protection is enabled. Deleting it removes that webhook configuration.
// It holds no data.
type InstallGuardOwner struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InstallGuardOwnerSpec   `json:"spec,omitempty"`
	Status InstallGuardOwnerStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// InstallGuardOwnerList contains a list of InstallGuardOwner
type InstallGuardOwnerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InstallGuardOwner `json:"items"`
}

func init() {
	SchemeBuilder.Register(&InstallGuardOwner{}, &InstallGuardOwnerList{})
}
