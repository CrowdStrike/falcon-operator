package v1alpha1

type FalconSecret struct {
	// Enable injecting sensitive Falcon values from existing k8s secret
	Enabled bool `json:"enabled"`
	// Namespace where the Falcon k8s secret is located.
	Namespace string `json:"namespace,omitempty"`
	// Name of the Falcon k8s secret
	SecretName string `json:"secretName,omitempty"`
}
