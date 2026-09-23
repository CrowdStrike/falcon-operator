package webhookcert

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"time"

	pkgtls "github.com/crowdstrike/falcon-operator/pkg/tls"
	"github.com/go-logr/logr"
	arv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	CertSecretName        = "webhook-server-cert"
	CertManagedLabel      = "falcon.crowdstrike.com/managed-cert"
	DefaultCertDir        = "/tmp/k8s-webhook-server/serving-certs"
	certRotationThreshold = 30 * 24 * time.Hour
	certValidityDays      = 3650
)

//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list

// ReconcileCert ensures the webhook TLS certificate is present and valid.
// It generates a self-signed cert if none exists, rotates managed certs near
// expiry, and writes the cert and key to certDir for the webhook server to read.
// The caBundle is also patched into the ValidatingWebhookConfiguration.
//
// When a new secret is created, it is owned by the operator Deployment so it is
// garbage-collected when the operator is uninstalled. Existing secrets are never
// given an owner reference — this preserves user-provided certs and secrets that
// survived a previous install.
//
// Custom cert support: if the secret already exists without the managed label
// it is treated as user-provided and will never be auto-rotated.
//
// OpenShift: when openShift is true and the secret has no ca.crt entry, the
// operator assumes OpenShift's service-signing CA is in use. It patches the
// webhook Service with service.beta.openshift.io/serving-cert-secret-name so
// OpenShift provisions the TLS cert automatically, and patches the
// ValidatingWebhookConfiguration with service.beta.openshift.io/inject-cabundle
// so OpenShift injects the CA bundle without manual caBundle management.
func ReconcileCert(ctx context.Context, cfg *rest.Config, scheme *runtime.Scheme, log logr.Logger, namespace, webhookConfigName, serviceName, certDir, secretName string, openShift bool) error {
	c, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		return fmt.Errorf("creating client for webhook cert reconciliation: %w", err)
	}

	ownerRef := lookupOwnerRef(ctx, c, log, namespace)

	certPEM, keyPEM, caBundle, err := getOrGenerateCert(ctx, c, log, namespace, serviceName, secretName, ownerRef)
	if err != nil {
		return err
	}

	if err := writeCerts(certDir, certPEM, keyPEM); err != nil {
		return err
	}

	if openShift && len(caBundle) == 0 {
		log.Info("OpenShift detected and no ca.crt in webhook cert secret; patching Service and ValidatingWebhookConfiguration with OpenShift service-signing CA annotations")
		if err := patchServiceOpenShift(ctx, c, log, namespace, serviceName); err != nil {
			return err
		}
		return patchWebhookOpenShift(ctx, c, webhookConfigName)
	}

	return patchWebhookCABundle(ctx, c, webhookConfigName, caBundle)
}

// OperatorNamespace returns the namespace the operator pod is running in.
// Reads POD_NAMESPACE env var first, then the in-cluster serviceaccount file,
// then falls back to the kustomize default.
func OperatorNamespace() string {
	if ns := os.Getenv("POD_NAMESPACE"); ns != "" {
		return ns
	}
	if data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace"); err == nil {
		return string(data)
	}
	return "falcon-operator-system"
}

// lookupOwnerRef finds the operator Deployment in the given namespace by the
// kubebuilder-standard label "control-plane: controller-manager" and returns
// an OwnerReference for it. Returns nil if the Deployment cannot be found;
// failures are logged but do not block cert reconciliation.
func lookupOwnerRef(ctx context.Context, c client.Client, log logr.Logger, namespace string) *metav1.OwnerReference {
	depList := &appsv1.DeploymentList{}
	if err := c.List(ctx, depList,
		client.InNamespace(namespace),
		client.MatchingLabels{"control-plane": "controller-manager"},
	); err != nil || len(depList.Items) == 0 {
		log.Info("could not look up operator Deployment; webhook-server-cert will not be auto-deleted on uninstall")
		return nil
	}
	dep := &depList.Items[0]
	t := true
	return &metav1.OwnerReference{
		APIVersion:         "apps/v1",
		Kind:               "Deployment",
		Name:               dep.Name,
		UID:                dep.UID,
		BlockOwnerDeletion: &t,
	}
}

func getOrGenerateCert(ctx context.Context, c client.Client, log logr.Logger, namespace, serviceName, secretName string, ownerRef *metav1.OwnerReference) (certPEM, keyPEM, caBundle []byte, err error) {
	secretKey := types.NamespacedName{Name: secretName, Namespace: namespace}
	existing := &corev1.Secret{}
	getErr := c.Get(ctx, secretKey, existing)

	switch {
	case apierrors.IsNotFound(getErr):
		log.Info("webhook-server-cert not found, generating self-signed cert")
		return generateAndStoreWebhookCert(ctx, c, namespace, serviceName, secretName, ownerRef)

	case getErr != nil:
		return nil, nil, nil, fmt.Errorf("getting webhook cert secret: %w", getErr)

	case existing.Labels[CertManagedLabel] == "true" && needsRotation(existing.Data["tls.crt"]):
		log.Info("webhook cert is near expiry, rotating")
		if err := c.Delete(ctx, existing); err != nil && !apierrors.IsNotFound(err) {
			return nil, nil, nil, fmt.Errorf("deleting expiring webhook cert: %w", err)
		}
		return generateAndStoreWebhookCert(ctx, c, namespace, serviceName, secretName, ownerRef)

	default:
		// Secret exists — either user-provided (no managed label) or not near expiry.
		// Do not modify ownership of existing secrets.
		return existing.Data["tls.crt"], existing.Data["tls.key"], existing.Data["ca.crt"], nil
	}
}

func generateAndStoreWebhookCert(ctx context.Context, c client.Client, namespace, serviceName, secretName string, ownerRef *metav1.OwnerReference) (certPEM, keyPEM, caBundle []byte, err error) {
	certInfo := pkgtls.CertInfo{
		CommonName: fmt.Sprintf("%s.%s.svc", serviceName, namespace),
		DNSNames: []string{
			fmt.Sprintf("%s.%s.svc", serviceName, namespace),
			fmt.Sprintf("%s.%s.svc.cluster.local", serviceName, namespace),
		},
	}

	certPEM, keyPEM, caBundle, err = pkgtls.CertSetup(namespace, certValidityDays, certInfo)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("generating webhook cert: %w", err)
	}

	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      secretName,
			Namespace: namespace,
			Labels:    map[string]string{CertManagedLabel: "true"},
		},
		Type: corev1.SecretTypeTLS,
		Data: map[string][]byte{
			"tls.crt": certPEM,
			"tls.key": keyPEM,
			"ca.crt":  caBundle,
		},
	}

	if ownerRef != nil {
		secret.OwnerReferences = []metav1.OwnerReference{*ownerRef}
	}

	if err := c.Create(ctx, secret); err != nil && !apierrors.IsAlreadyExists(err) {
		return nil, nil, nil, fmt.Errorf("creating webhook cert secret: %w", err)
	}

	return certPEM, keyPEM, caBundle, nil
}

// writeCerts writes the TLS cert and key to certDir for the webhook server.
func writeCerts(certDir string, certPEM, keyPEM []byte) error {
	if err := os.MkdirAll(certDir, 0755); err != nil {
		return fmt.Errorf("creating cert directory %q: %w", certDir, err)
	}
	if err := os.WriteFile(filepath.Join(certDir, "tls.crt"), certPEM, 0644); err != nil {
		return fmt.Errorf("writing tls.crt: %w", err)
	}
	if err := os.WriteFile(filepath.Join(certDir, "tls.key"), keyPEM, 0600); err != nil {
		return fmt.Errorf("writing tls.key: %w", err)
	}
	return nil
}

// needsRotation returns true if the PEM-encoded cert expires within the rotation threshold.
func needsRotation(certPEM []byte) bool {
	if len(certPEM) == 0 {
		return true
	}
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return true
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return true
	}
	return time.Until(cert.NotAfter) < certRotationThreshold
}

// patchWebhookCABundle sets caBundle on every webhook in the named
// ValidatingWebhookConfiguration. If the resource does not exist yet
// (manifests not yet applied) the call is a no-op.
func patchWebhookCABundle(ctx context.Context, c client.Client, webhookConfigName string, caBundle []byte) error {
	webhookConfig := &arv1.ValidatingWebhookConfiguration{}
	if err := c.Get(ctx, types.NamespacedName{Name: webhookConfigName}, webhookConfig); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("getting ValidatingWebhookConfiguration %q: %w", webhookConfigName, err)
	}

	updated := webhookConfig.DeepCopy()
	for i := range updated.Webhooks {
		updated.Webhooks[i].ClientConfig.CABundle = caBundle
	}

	if err := c.Update(ctx, updated); err != nil {
		return fmt.Errorf("updating webhook caBundle: %w", err)
	}
	return nil
}

// patchServiceOpenShift adds the service.beta.openshift.io/serving-cert-secret-name
// annotation to the webhook Service so OpenShift's service-signing CA provisions
// the TLS cert Secret automatically. If the Service does not exist the call is a no-op.
func patchServiceOpenShift(ctx context.Context, c client.Client, log logr.Logger, namespace, serviceName string) error {
	svc := &corev1.Service{}
	if err := c.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: namespace}, svc); err != nil {
		if apierrors.IsNotFound(err) {
			log.Info("webhook Service not found; skipping OpenShift serving-cert annotation", "service", serviceName)
			return nil
		}
		return fmt.Errorf("getting webhook Service %q: %w", serviceName, err)
	}

	const annotation = "service.beta.openshift.io/serving-cert-secret-name"
	if svc.Annotations[annotation] == CertSecretName {
		return nil
	}

	updated := svc.DeepCopy()
	if updated.Annotations == nil {
		updated.Annotations = make(map[string]string)
	}
	updated.Annotations[annotation] = CertSecretName

	if err := c.Update(ctx, updated); err != nil {
		return fmt.Errorf("patching webhook Service with OpenShift serving-cert annotation: %w", err)
	}
	return nil
}

// patchWebhookOpenShift adds the service.beta.openshift.io/inject-cabundle annotation
// to the ValidatingWebhookConfiguration so OpenShift injects the service-signing CA
// bundle automatically. If the resource does not exist the call is a no-op.
func patchWebhookOpenShift(ctx context.Context, c client.Client, webhookConfigName string) error {
	webhookConfig := &arv1.ValidatingWebhookConfiguration{}
	if err := c.Get(ctx, types.NamespacedName{Name: webhookConfigName}, webhookConfig); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("getting ValidatingWebhookConfiguration %q: %w", webhookConfigName, err)
	}

	const annotation = "service.beta.openshift.io/inject-cabundle"
	if webhookConfig.Annotations[annotation] == "true" {
		return nil
	}

	updated := webhookConfig.DeepCopy()
	if updated.Annotations == nil {
		updated.Annotations = make(map[string]string)
	}
	updated.Annotations[annotation] = "true"

	if err := c.Update(ctx, updated); err != nil {
		return fmt.Errorf("patching ValidatingWebhookConfiguration with OpenShift inject-cabundle annotation: %w", err)
	}
	return nil
}
