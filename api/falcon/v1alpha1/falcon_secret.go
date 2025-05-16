package v1alpha1

// FalconSecret configures injecting Falcon secrets from an existing k8s secret.
// The k8s secret must have already been created in your cluster, before you enable this option.
type FalconSecret struct {
	// Enable injecting sensitive Falcon values from existing k8s secret
	// +kubebuilder:default=false
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Secret Enabled",order=1
	Enabled bool `json:"enabled"`
	// Namespace where the Falcon k8s secret is located.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Secret Namespace",order=2
	Namespace string `json:"namespace,omitempty"`
	// SecretName of the existing Falcon k8s secret
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Secret SecretName",order=3
	SecretName string `json:"secretName,omitempty"`
}
