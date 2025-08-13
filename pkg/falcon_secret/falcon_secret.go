package falcon_secret

import (
	"strings"

	corev1 "k8s.io/api/core/v1"
)

func GetFalconCredsFromSecret(secret *corev1.Secret) (clientId, clientSecret string) {
	if clientIdFromSecret, exists := secret.Data["falcon-client-id"]; exists {
		clientId = strings.TrimSpace(string(clientIdFromSecret))
	}

	if clientSecretFromSecret, exists := secret.Data["falcon-client-secret"]; exists {
		clientSecret = strings.TrimSpace(string(clientSecretFromSecret))
	}

	return clientId, clientSecret
}

func GetFalconCIDFromSecret(secret *corev1.Secret) (cid *string) {
	if cidFromSecret, exists := secret.Data["falcon-cid"]; exists {
		trimmedCID := strings.TrimSpace(string(cidFromSecret))
		if trimmedCID == "" {
			return nil
		}
		cid = &trimmedCID
	}

	return cid
}

func GetFalconProvisioningTokenFromSecret(secret *corev1.Secret) (provisioningToken string) {
	if provisioningTokenFromSecret, exists := secret.Data["falcon-provisioning-token"]; exists {
		provisioningToken = strings.TrimSpace(string(provisioningTokenFromSecret))
	}

	return provisioningToken
}
