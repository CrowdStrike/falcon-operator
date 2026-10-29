package webhookcert

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	pkgtls "github.com/crowdstrike/falcon-operator/pkg/tls"
	arv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func newFakeClient(objs ...client.Object) client.Client {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = arv1.AddToScheme(scheme)
	return fake.NewClientBuilder().WithScheme(scheme).WithObjects(objs...).Build()
}

func webhookService(namespace, name string, annotations map[string]string) *corev1.Service {
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: annotations,
		},
	}
}

func validatingWebhookConfig(name string, annotations map[string]string) *arv1.ValidatingWebhookConfiguration {
	return &arv1.ValidatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Annotations: annotations,
		},
	}
}

// ── getOrGenerateCert ──────────────────────────────────────────────────────────

// TestGetOrGenerateCert_UserProvidedSecret verifies that when a Secret exists
// without the managed label, getOrGenerateCert uses it as-is — simulating the
// "operator restart after user pre-creates cert" path.
func TestGetOrGenerateCert_UserProvidedSecret(t *testing.T) {
	cert, key, ca, err := pkgtls.CertSetup("test-ns", 365, pkgtls.CertInfo{
		CommonName: "webhook-service.test-ns.svc",
		DNSNames:   []string{"webhook-service.test-ns.svc"},
	})
	if err != nil {
		t.Fatalf("generating test cert: %v", err)
	}

	// Pre-create secret without the managed label, as a user would.
	userSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      CertSecretName,
			Namespace: "test-ns",
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{"tls.crt": cert, "tls.key": key, "ca.crt": ca},
	}

	c := newFakeClient(userSecret)
	log := zap.New()

	gotCert, gotKey, gotCA, err := getOrGenerateCert(context.Background(), c, log, "test-ns", "webhook-service", CertSecretName, nil)
	if err != nil {
		t.Fatalf("getOrGenerateCert: %v", err)
	}

	if !bytes.Equal(gotCert, cert) {
		t.Error("returned cert does not match user-provided cert")
	}
	if !bytes.Equal(gotKey, key) {
		t.Error("returned key does not match user-provided key")
	}
	if !bytes.Equal(gotCA, ca) {
		t.Error("returned CA does not match user-provided CA")
	}

	// The secret must not have been modified (no managed label added).
	got := &corev1.Secret{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: CertSecretName, Namespace: "test-ns"}, got); err != nil {
		t.Fatalf("getting secret after reconcile: %v", err)
	}
	if got.Labels[CertManagedLabel] == "true" {
		t.Error("user-provided secret must not be given the managed label")
	}
}

func TestGetOrGenerateCert_UserProvidedSecret_WritesToDisk(t *testing.T) {
	cert, key, ca, err := pkgtls.CertSetup("test-ns", 365, pkgtls.CertInfo{
		CommonName: "webhook-service.test-ns.svc",
		DNSNames:   []string{"webhook-service.test-ns.svc"},
	})
	if err != nil {
		t.Fatalf("generating test cert: %v", err)
	}

	userSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: CertSecretName, Namespace: "test-ns"},
		Type:       corev1.SecretTypeTLS,
		Data:       map[string][]byte{"tls.crt": cert, "tls.key": key, "ca.crt": ca},
	}

	c := newFakeClient(userSecret)
	log := zap.New()

	gotCert, gotKey, _, err := getOrGenerateCert(context.Background(), c, log, "test-ns", "webhook-service", CertSecretName, nil)
	if err != nil {
		t.Fatalf("getOrGenerateCert: %v", err)
	}

	certDir := t.TempDir()
	if err := writeCerts(certDir, gotCert, gotKey); err != nil {
		t.Fatalf("writeCerts: %v", err)
	}

	diskCert, err := os.ReadFile(filepath.Join(certDir, "tls.crt"))
	if err != nil {
		t.Fatalf("reading tls.crt from disk: %v", err)
	}
	diskKey, err := os.ReadFile(filepath.Join(certDir, "tls.key"))
	if err != nil {
		t.Fatalf("reading tls.key from disk: %v", err)
	}

	if !bytes.Equal(diskCert, cert) {
		t.Error("tls.crt on disk does not match user-provided cert")
	}
	if !bytes.Equal(diskKey, key) {
		t.Error("tls.key on disk does not match user-provided key")
	}
}

func TestGetOrGenerateCert_GeneratesWhenAbsent(t *testing.T) {
	c := newFakeClient()
	log := zap.New()

	cert, key, ca, err := getOrGenerateCert(context.Background(), c, log, "test-ns", "webhook-service", CertSecretName, nil)
	if err != nil {
		t.Fatalf("getOrGenerateCert: %v", err)
	}
	if len(cert) == 0 || len(key) == 0 || len(ca) == 0 {
		t.Error("expected non-empty cert/key/ca to be generated")
	}

	// Secret must have been created with the managed label.
	got := &corev1.Secret{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: CertSecretName, Namespace: "test-ns"}, got); err != nil {
		t.Fatalf("getting generated secret: %v", err)
	}
	if got.Labels[CertManagedLabel] != "true" {
		t.Error("generated secret must have the managed label")
	}
}

// ── patchServiceOpenShift ──────────────────────────────────────────────────────

func TestPatchServiceOpenShift_ServiceNotFound(t *testing.T) {
	c := newFakeClient()
	log := zap.New()
	err := patchServiceOpenShift(context.Background(), c, log, "test-ns", "webhook-service")
	if err != nil {
		t.Errorf("expected no error when Service not found, got %v", err)
	}
}

func TestPatchServiceOpenShift_AnnotationAlreadySet(t *testing.T) {
	svc := webhookService("test-ns", "webhook-service", map[string]string{
		"service.beta.openshift.io/serving-cert-secret-name": CertSecretName,
	})
	c := newFakeClient(svc)
	log := zap.New()

	if err := patchServiceOpenShift(context.Background(), c, log, "test-ns", "webhook-service"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Confirm no spurious update happened (resourceVersion unchanged from initial empty string).
	got := &corev1.Service{}
	_ = c.Get(context.Background(), types.NamespacedName{Name: "webhook-service", Namespace: "test-ns"}, got)
	if got.Annotations["service.beta.openshift.io/serving-cert-secret-name"] != CertSecretName {
		t.Error("annotation should still be set")
	}
}

func TestPatchServiceOpenShift_AnnotationAdded(t *testing.T) {
	svc := webhookService("test-ns", "webhook-service", nil)
	c := newFakeClient(svc)
	log := zap.New()

	if err := patchServiceOpenShift(context.Background(), c, log, "test-ns", "webhook-service"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := &corev1.Service{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: "webhook-service", Namespace: "test-ns"}, got); err != nil {
		t.Fatalf("getting updated Service: %v", err)
	}
	want := CertSecretName
	if got.Annotations["service.beta.openshift.io/serving-cert-secret-name"] != want {
		t.Errorf("expected annotation %q, got %q", want, got.Annotations["service.beta.openshift.io/serving-cert-secret-name"])
	}
}

// ── patchWebhookOpenShift ──────────────────────────────────────────────────────

func TestPatchWebhookOpenShift_NotFound(t *testing.T) {
	c := newFakeClient()
	err := patchWebhookOpenShift(context.Background(), c, "validating-webhook-configuration")
	if err != nil {
		t.Errorf("expected no error when VWC not found, got %v", err)
	}
}

func TestPatchWebhookOpenShift_AnnotationAlreadySet(t *testing.T) {
	vwc := validatingWebhookConfig("validating-webhook-configuration", map[string]string{
		"service.beta.openshift.io/inject-cabundle": "true",
	})
	c := newFakeClient(vwc)

	if err := patchWebhookOpenShift(context.Background(), c, "validating-webhook-configuration"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := &arv1.ValidatingWebhookConfiguration{}
	_ = c.Get(context.Background(), types.NamespacedName{Name: "validating-webhook-configuration"}, got)
	if got.Annotations["service.beta.openshift.io/inject-cabundle"] != "true" {
		t.Error("annotation should still be set")
	}
}

func TestPatchWebhookOpenShift_AnnotationAdded(t *testing.T) {
	vwc := validatingWebhookConfig("validating-webhook-configuration", nil)
	c := newFakeClient(vwc)

	if err := patchWebhookOpenShift(context.Background(), c, "validating-webhook-configuration"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := &arv1.ValidatingWebhookConfiguration{}
	if err := c.Get(context.Background(), types.NamespacedName{Name: "validating-webhook-configuration"}, got); err != nil {
		t.Fatalf("getting updated VWC: %v", err)
	}
	if got.Annotations["service.beta.openshift.io/inject-cabundle"] != "true" {
		t.Errorf("expected inject-cabundle=true, got %q", got.Annotations["service.beta.openshift.io/inject-cabundle"])
	}
}
