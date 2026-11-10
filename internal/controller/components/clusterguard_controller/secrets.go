package clusterguard_controller

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"slices"

	"github.com/crowdstrike/falcon-operator/internal/controller/assets"
	k8sutils "github.com/crowdstrike/falcon-operator/internal/controller/common"
	pkgcommon "github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/crowdstrike/falcon-operator/pkg/tls"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

// webhookHost returns the hostname the API server uses to reach the admission webhook: the Service DNS name, with
// spec.controller.domainName appended when it is set.
func (a *ClusterGuardController) webhookHost() string {
	host := fmt.Sprintf("%s.%s.svc", pkgcommon.ClusterGuardControllerWebhookServiceName, a.cfg.InstallNamespace)
	if a.cfg.ClusterGuardControllerConfig.DomainName != "" {
		host = fmt.Sprintf("%s.%s", host, a.cfg.ClusterGuardControllerConfig.DomainName)
	}
	return host
}

// webhookDNSNames returns the admission webhook certificate's DNS names. The first entry is the certificate's common
// name.
func (a *ClusterGuardController) webhookDNSNames() []string {
	host := a.webhookHost()
	return []string{
		host,
		fmt.Sprintf("%s.cluster.local", host),
		fmt.Sprintf("%s.%s", pkgcommon.ClusterGuardControllerWebhookServiceName, a.cfg.InstallNamespace),
		pkgcommon.ClusterGuardControllerWebhookServiceName,
	}
}

// reconcileTLSSecret reconciles the TLS secret for the admission webhook. The secret is regenerated when its
// certificate does not carry the desired DNS names, e.g. after spec.controller.domainName changes.
func (a *ClusterGuardController) reconcileTLSSecret(ctx context.Context) (*corev1.Secret, error) {
	existing := &corev1.Secret{}
	namespace := a.cfg.InstallNamespace
	err := pkgcommon.GetNamespacedObject(ctx, a.r, a.r.GetK8sReader(), types.NamespacedName{Name: pkgcommon.ClusterGuardControllerTLSSecretName, Namespace: namespace}, existing)
	if err != nil && !apierrors.IsNotFound(err) {
		a.r.GetLog().Error(err, "Failed to get FalconClusterGuard TLS Secret")
		return &corev1.Secret{}, err
	}
	found := err == nil

	dnsNames := a.webhookDNSNames()
	if found && certHasDNSNames(existing.Data["tls.crt"], dnsNames) {
		return existing, nil
	}

	validity := 3650
	if a.cfg.ClusterGuardControllerConfig.TLS.Validity != nil {
		validity = *a.cfg.ClusterGuardControllerConfig.TLS.Validity
	}
	cert, key, ca, err := tls.CertSetup(namespace, validity, tls.CertInfo{CommonName: dnsNames[0], DNSNames: dnsNames})
	if err != nil {
		a.r.GetLog().Error(err, "Failed to generate FalconClusterGuard TLS certificates")
		return &corev1.Secret{}, err
	}
	data := map[string][]byte{"tls.crt": cert, "tls.key": key, "ca.crt": ca}

	if found {
		a.r.GetLog().Info("Regenerating the FalconClusterGuard webhook TLS Secret: certificate DNS names changed")
		existing.Data = data
		existing.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Secret"))
		if err := k8sutils.Update(a.r, ctx, a.cfg.Request, a.r.GetLog(), a.cfg.Owner, a.cfg.Status, existing); err != nil {
			return &corev1.Secret{}, err
		}
		return existing, nil
	}

	secretLabels := pkgcommon.CRLabels("secret", pkgcommon.ClusterGuardControllerTLSSecretName, pkgcommon.ClusterGuardControllerComponentName)
	secretLabels["app"] = pkgcommon.ClusterGuardServiceApp
	a.withAppLabels(secretLabels)
	tlsSecret := assets.SecretWithCustomLabels(pkgcommon.ClusterGuardControllerTLSSecretName, namespace, data, corev1.SecretTypeTLS, secretLabels)
	if err := k8sutils.Create(a.r, a.r.GetScheme(), ctx, a.cfg.Request, a.r.GetLog(), a.cfg.Owner, a.cfg.Status, tlsSecret); err != nil {
		return &corev1.Secret{}, err
	}
	return tlsSecret, nil
}

// certHasDNSNames reports whether the PEM-encoded certificate carries exactly dnsNames, in order.
func certHasDNSNames(certPEM []byte, dnsNames []string) bool {
	block, _ := pem.Decode(certPEM)
	if block == nil {
		return false
	}
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		return false
	}
	return slices.Equal(cert.DNSNames, dnsNames)
}
