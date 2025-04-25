package common

import (
	corev1 "k8s.io/api/core/v1"
	"strings"
)

func GetFalconCredsFromSecret(secret *corev1.Secret) (clientId, clientSecret, cid string) {
	if clientIdFromSecret, exists := secret.Data["FalconClientId"]; exists {
		clientId = strings.ReplaceAll(string(clientIdFromSecret), "\n", "")
	}

	if clientSecretFromSecret, exists := secret.Data["FalconClientSecret"]; exists {
		clientSecret = strings.ReplaceAll(string(clientSecretFromSecret), "\n", "")
	}

	if cidFromSecret, exists := secret.Data["FalconCID"]; exists {
		cid = strings.ReplaceAll(string(cidFromSecret), "\n", "")
	}

	return clientId, clientSecret, cid
}

func GetFalconProvisioningTokenFromSecret(secret *corev1.Secret) (provisioningToken string) {
	if provisioningTokenFromSecret, exists := secret.Data["FalconProvisioningToken"]; exists {
		provisioningToken = strings.ReplaceAll(string(provisioningTokenFromSecret), "\n", "")
	}

	return provisioningToken
}
