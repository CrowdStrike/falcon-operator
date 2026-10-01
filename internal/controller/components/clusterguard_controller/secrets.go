package clusterguard_controller

import (
	"context"
	"fmt"

	"github.com/crowdstrike/falcon-operator/internal/controller/assets"
	k8sutils "github.com/crowdstrike/falcon-operator/internal/controller/common"
	pkgcommon "github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/crowdstrike/falcon-operator/pkg/tls"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

// reconcileTLSSecret reconciles the TLS secret for the admission webhook.
func (a *ClusterGuardController) reconcileTLSSecret(ctx context.Context) (*corev1.Secret, error) {
	existing := &corev1.Secret{}
	namespace := a.cfg.InstallNamespace
	err := pkgcommon.GetNamespacedObject(ctx, a.r, a.r.GetK8sReader(), types.NamespacedName{Name: pkgcommon.AdmissionTLSSecretName, Namespace: namespace}, existing)
	if err != nil && apierrors.IsNotFound(err) {
		svcName := fmt.Sprintf("%s.%s.svc", pkgcommon.AdmissionWebhookServiceName, namespace)
		altDNSNames := []string{
			svcName,
			fmt.Sprintf("%s.cluster.local", svcName),
			fmt.Sprintf("%s.%s", svcName, namespace),
		}
		validity := 3650
		if a.cfg.ClusterGuardControllerConfig.TLS.Validity != nil {
			validity = *a.cfg.ClusterGuardControllerConfig.TLS.Validity
		}
		cert, key, ca, err := tls.CertSetup(namespace, validity, tls.CertInfo{CommonName: svcName, DNSNames: altDNSNames})
		if err != nil {
			a.r.GetLog().Error(err, "Failed to generate FalconClusterGuard TLS certificates")
			return &corev1.Secret{}, err
		}
		secretLabels := pkgcommon.CRLabels("secret", pkgcommon.AdmissionTLSSecretName, pkgcommon.AdmissionComponentName)
		secretLabels["app"] = pkgcommon.AdmissionServiceApp
		tlsSecret := assets.SecretWithCustomLabels(pkgcommon.AdmissionTLSSecretName, namespace,
			map[string][]byte{"tls.crt": cert, "tls.key": key, "ca.crt": ca}, corev1.SecretTypeTLS, secretLabels)
		if err := k8sutils.Create(a.r, a.r.GetScheme(), ctx, a.cfg.Request, a.r.GetLog(), a.cfg.Owner, a.cfg.Status, tlsSecret); err != nil {
			return &corev1.Secret{}, err
		}
		return tlsSecret, nil
	} else if err != nil {
		a.r.GetLog().Error(err, "Failed to get FalconClusterGuard TLS Secret")
		return &corev1.Secret{}, err
	}
	return existing, nil
}
