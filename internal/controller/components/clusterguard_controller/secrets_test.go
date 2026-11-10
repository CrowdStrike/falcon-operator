package clusterguard_controller

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"slices"
	"testing"

	pkgcommon "github.com/crowdstrike/falcon-operator/pkg/common"
	arv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func certDNSNames(t *testing.T, certPEM []byte) []string {
	t.Helper()
	block, _ := pem.Decode(certPEM)
	if block == nil {
		t.Fatal("failed to decode tls.crt")
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		t.Fatal(err)
	}
	return cert.DNSNames
}

func TestReconcileTLSSecretDNSNames(t *testing.T) {
	tests := []struct {
		name       string
		domainName string
		want       []string
	}{
		{
			name: "default",
			want: []string{
				"webhook.falcon-clusterguard.svc",
				"webhook.falcon-clusterguard.svc.cluster.local",
				"webhook.falcon-clusterguard",
				"webhook",
			},
		},
		{
			name:       "domainName",
			domainName: "testing.io",
			want: []string{
				"webhook.falcon-clusterguard.svc.testing.io",
				"webhook.falcon-clusterguard.svc.testing.io.cluster.local",
				"webhook.falcon-clusterguard",
				"webhook",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, a := newCleanupTestController(t)
			a.cfg.ClusterGuardControllerConfig.DomainName = tt.domainName

			s, err := a.reconcileTLSSecret(context.Background())
			if err != nil {
				t.Fatalf("reconcileTLSSecret() error: %v", err)
			}
			if got := certDNSNames(t, s.Data["tls.crt"]); !slices.Equal(got, tt.want) {
				t.Errorf("DNSNames = %v, want %v", got, tt.want)
			}
		})
	}
}

// TestReconcileTLSSecretRegeneratesOnDNSNameChange also checks that the new CA and URL reach the
// ValidatingWebhookConfiguration, whose update is what rolls the Deployment onto the new certificate.
func TestReconcileTLSSecretRegeneratesOnDNSNameChange(t *testing.T) {
	ctx := context.Background()
	r, _, a := newCleanupTestController(t)
	stored := func() *corev1.Secret {
		t.Helper()
		s := &corev1.Secret{}
		if err := r.Get(ctx, types.NamespacedName{Name: pkgcommon.ClusterGuardControllerTLSSecretName, Namespace: cleanupTestNamespace}, s); err != nil {
			t.Fatal(err)
		}
		return s
	}

	if _, err := a.reconcileTLSSecret(ctx); err != nil {
		t.Fatalf("reconcileTLSSecret() error: %v", err)
	}
	ca := stored().Data["ca.crt"]
	if _, err := a.reconcileValidatingWebhook(ctx, ca); err != nil {
		t.Fatalf("reconcileValidatingWebhook() error: %v", err)
	}

	if _, err := a.reconcileTLSSecret(ctx); err != nil {
		t.Fatalf("reconcileTLSSecret() error: %v", err)
	}
	if string(stored().Data["ca.crt"]) != string(ca) {
		t.Error("a certificate with the desired DNS names was regenerated")
	}

	a.cfg.ClusterGuardControllerConfig.DomainName = "testing.io"
	returned, err := a.reconcileTLSSecret(ctx)
	if err != nil {
		t.Fatalf("reconcileTLSSecret() error: %v", err)
	}
	s := stored()
	if string(s.Data["ca.crt"]) == string(ca) {
		t.Error("changing domainName did not regenerate the certificate")
	}
	if string(returned.Data["ca.crt"]) != string(s.Data["ca.crt"]) {
		t.Error("the returned Secret does not match the stored one")
	}
	if got := certDNSNames(t, s.Data["tls.crt"]); got[0] != "webhook.falcon-clusterguard.svc.testing.io" {
		t.Errorf("DNSNames = %v, want the domainName host first", got)
	}

	updated, err := a.reconcileValidatingWebhook(ctx, returned.Data["ca.crt"])
	if err != nil {
		t.Fatalf("reconcileValidatingWebhook() error: %v", err)
	}
	if !updated {
		t.Error("the ValidatingWebhookConfiguration was not updated for the new CA and URL")
	}
	vwc := &arv1.ValidatingWebhookConfiguration{}
	if err := r.Get(ctx, types.NamespacedName{Name: pkgcommon.ClusterGuardControllerValidatingWebhookName}, vwc); err != nil {
		t.Fatal(err)
	}
	for _, w := range vwc.Webhooks {
		if string(w.ClientConfig.CABundle) != string(s.Data["ca.crt"]) || w.ClientConfig.URL == nil {
			t.Errorf("%s: clientConfig not updated: %+v", w.Name, w.ClientConfig)
		}
	}
}
