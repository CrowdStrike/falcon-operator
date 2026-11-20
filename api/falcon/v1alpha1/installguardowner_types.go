package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster

// InstallGuardOwner is the owner of the FCG install guard ValidatingWebhookConfiguration created by the
// clusterguard controller when uninstall protection is enabled. Deleting it removes that webhook configuration.
// It holds no data.
type InstallGuardOwner struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
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
