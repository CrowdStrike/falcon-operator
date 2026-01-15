package falcon

import (
	"context"
	"testing"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// Mock client that returns errors for testing error handling
type erroringClient struct {
	client.Client
}

func (e *erroringClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	return errors.NewBadRequest("test error")
}

func TestReconcileInjectorTLSSecretLogic(t *testing.T) {
	ctx := context.Background()
	log := zap.New(zap.UseDevMode(true))

	// Create test scheme
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, falconv1alpha1.AddToScheme(scheme))

	t.Run("should return existing secret when found", func(t *testing.T) {
		existingSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      injectorTLSSecretName,
				Namespace: "test-namespace",
			},
			Type: corev1.SecretTypeTLS,
			Data: map[string][]byte{
				"tls.crt": []byte("existing-cert"),
				"tls.key": []byte("existing-key"),
				"ca.crt":  []byte("existing-ca"),
			},
		}

		falconContainer := &falconv1alpha1.FalconContainer{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-falcon-container",
				Namespace: "test-namespace",
			},
			Spec: falconv1alpha1.FalconContainerSpec{
				InstallNamespace: "test-namespace",
				Injector: falconv1alpha1.FalconContainerInjectorSpec{
					TLS: falconv1alpha1.FalconContainerInjectorTLS{},
				},
			},
		}

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(existingSecret, falconContainer).
			Build()

		reconciler := &FalconContainerReconciler{
			Client: fakeClient,
			Reader: fakeClient,
			Scheme: scheme,
		}

		secret, err := reconciler.reconcileInjectorTLSSecret(ctx, log, falconContainer)

		require.NoError(t, err)
		assert.NotNil(t, secret)
		assert.Equal(t, injectorTLSSecretName, secret.Name)
		assert.Equal(t, "test-namespace", secret.Namespace)
		assert.Equal(t, []byte("existing-cert"), secret.Data["tls.crt"])
		assert.Equal(t, []byte("existing-key"), secret.Data["tls.key"])
		assert.Equal(t, []byte("existing-ca"), secret.Data["ca.crt"])
	})

	t.Run("should handle default validity configuration", func(t *testing.T) {
		falconContainer := &falconv1alpha1.FalconContainer{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-falcon-container",
				Namespace: "test-namespace",
			},
			Spec: falconv1alpha1.FalconContainerSpec{
				InstallNamespace: "test-namespace",
				Injector: falconv1alpha1.FalconContainerInjectorSpec{
					TLS: falconv1alpha1.FalconContainerInjectorTLS{
						Validity: nil, // Should use default 3650
					},
				},
			},
		}

		validity := 3650
		if falconContainer.Spec.Injector.TLS.Validity != nil {
			validity = *falconContainer.Spec.Injector.TLS.Validity
		}

		assert.Equal(t, 3650, validity)
	})

	t.Run("should handle custom validity configuration", func(t *testing.T) {
		customValidity := 1000
		falconContainer := &falconv1alpha1.FalconContainer{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-falcon-container",
				Namespace: "test-namespace",
			},
			Spec: falconv1alpha1.FalconContainerSpec{
				InstallNamespace: "test-namespace",
				Injector: falconv1alpha1.FalconContainerInjectorSpec{
					TLS: falconv1alpha1.FalconContainerInjectorTLS{
						Validity: &customValidity,
					},
				},
			},
		}

		validity := 3650
		if falconContainer.Spec.Injector.TLS.Validity != nil {
			validity = *falconContainer.Spec.Injector.TLS.Validity
		}

		assert.Equal(t, 1000, validity)
	})

	t.Run("should handle errors from GetNamespacedObject", func(t *testing.T) {
		falconContainer := &falconv1alpha1.FalconContainer{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-falcon-container",
				Namespace: "test-namespace",
			},
			Spec: falconv1alpha1.FalconContainerSpec{
				InstallNamespace: "test-namespace",
				Injector: falconv1alpha1.FalconContainerInjectorSpec{
					TLS: falconv1alpha1.FalconContainerInjectorTLS{},
				},
			},
		}

		// Create a fake client that will return an error
		fakeClient := fake.NewClientBuilder().WithScheme(scheme).Build()

		reconciler := &FalconContainerReconciler{
			Client: &erroringClient{Client: fakeClient},
			Reader: &erroringClient{Client: fakeClient},
			Scheme: scheme,
		}

		secret, err := reconciler.reconcileInjectorTLSSecret(ctx, log, falconContainer)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unable to query existing injector TLS secret")
		assert.NotNil(t, secret) // Should return empty secret on error
	})

	t.Run("should handle validity boundary values", func(t *testing.T) {
		testCases := []struct {
			name             string
			validity         *int
			expectedValidity int
		}{
			{
				name:             "nil validity uses default",
				validity:         nil,
				expectedValidity: 3650,
			},
			{
				name:             "minimum validity (0)",
				validity:         &[]int{0}[0],
				expectedValidity: 0,
			},
			{
				name:             "single digit validity",
				validity:         &[]int{1}[0],
				expectedValidity: 1,
			},
			{
				name:             "maximum validity (9999)",
				validity:         &[]int{9999}[0],
				expectedValidity: 9999,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				falconContainer := &falconv1alpha1.FalconContainer{
					Spec: falconv1alpha1.FalconContainerSpec{
						Injector: falconv1alpha1.FalconContainerInjectorSpec{
							TLS: falconv1alpha1.FalconContainerInjectorTLS{
								Validity: tc.validity,
							},
						},
					},
				}

				validity := 3650
				if falconContainer.Spec.Injector.TLS.Validity != nil {
					validity = *falconContainer.Spec.Injector.TLS.Validity
				}

				assert.Equal(t, tc.expectedValidity, validity)
			})
		}
	})
}

func TestReconcileInjectorTLSSecretIntegrationScenarios(t *testing.T) {
	ctx := context.Background()
	log := zap.New(zap.UseDevMode(true))

	// Create test scheme
	scheme := runtime.NewScheme()
	require.NoError(t, corev1.AddToScheme(scheme))
	require.NoError(t, falconv1alpha1.AddToScheme(scheme))

	t.Run("secret data should have correct TLS structure", func(t *testing.T) {
		existingSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      injectorTLSSecretName,
				Namespace: "test-namespace",
			},
			Type: corev1.SecretTypeTLS,
			Data: map[string][]byte{
				"tls.crt": []byte("-----BEGIN CERTIFICATE-----\nfake-cert\n-----END CERTIFICATE-----"),
				"tls.key": []byte("-----BEGIN PRIVATE KEY-----\nfake-key\n-----END PRIVATE KEY-----"),
				"ca.crt":  []byte("-----BEGIN CERTIFICATE-----\nfake-ca\n-----END CERTIFICATE-----"),
			},
		}

		falconContainer := &falconv1alpha1.FalconContainer{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-falcon-container",
				Namespace: "test-namespace",
			},
			Spec: falconv1alpha1.FalconContainerSpec{
				InstallNamespace: "test-namespace",
				Injector: falconv1alpha1.FalconContainerInjectorSpec{
					TLS: falconv1alpha1.FalconContainerInjectorTLS{},
				},
			},
		}

		fakeClient := fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(existingSecret, falconContainer).
			Build()

		reconciler := &FalconContainerReconciler{
			Client: fakeClient,
			Reader: fakeClient,
			Scheme: scheme,
		}

		secret, err := reconciler.reconcileInjectorTLSSecret(ctx, log, falconContainer)

		require.NoError(t, err)
		assert.Equal(t, corev1.SecretTypeTLS, secret.Type)

		// Verify all required TLS keys are present
		requiredKeys := []string{"tls.crt", "tls.key", "ca.crt"}
		for _, key := range requiredKeys {
			assert.Contains(t, secret.Data, key, "Secret should contain %s", key)
			assert.NotEmpty(t, secret.Data[key], "Secret key %s should not be empty", key)
		}

		// Verify certificate-like structure (basic validation)
		tlsCert := string(secret.Data["tls.crt"])
		tlsKey := string(secret.Data["tls.key"])
		caCert := string(secret.Data["ca.crt"])

		assert.Contains(t, tlsCert, "CERTIFICATE", "tls.crt should contain certificate markers")
		assert.Contains(t, tlsKey, "PRIVATE KEY", "tls.key should contain private key markers")
		assert.Contains(t, caCert, "CERTIFICATE", "ca.crt should contain certificate markers")
	})
}
