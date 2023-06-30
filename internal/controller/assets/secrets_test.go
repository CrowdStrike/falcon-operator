package assets

import (
	"testing"

	"github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestSecretDockerConfigJson tests the secret function with the type DockerConfigJson
func TestSecretDockerConfigJson(t *testing.T) {
	dockerConfigSecret := map[string][]byte{
		corev1.DockerConfigJsonKey: []byte(`{"auths":{"test":{"username":"test","password":"test","auth":"dGVzdDp0ZXN0"}}}`),
	}
	want := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
			Labels:    common.CRLabels("secret", "test", "test"),
		},
		Type: corev1.SecretTypeDockerConfigJson,
		Data: dockerConfigSecret,
	}

	got := Secret("test", "test", "test", dockerConfigSecret, corev1.SecretTypeDockerConfigJson)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("SecretDockerConfigJson() mismatch (-want +got): %s", diff)
	}
}

// TestSecretTLS tests the secret function with the type TLS
func TestSecretTLS(t *testing.T) {
	tlsData := map[string][]byte{
		"tls.crt": []byte("test"),
		"tls.key": []byte("test"),
		"ca.crt":  []byte("test"),
	}
	want := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
			Labels:    common.CRLabels("secret", "test", "test"),
		},
		Type: corev1.SecretTypeTLS,
		Data: tlsData,
	}

	got := Secret("test", "test", "test", tlsData, corev1.SecretTypeTLS)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("SecretTLS() mismatch (-want +got): %s", diff)
	}
}
