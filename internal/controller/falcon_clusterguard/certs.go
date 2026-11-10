package controllers

import (
	"context"
	"crypto/x509"
	"fmt"
	"maps"
	"net/url"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/internal/controller/assets"
	k8sutils "github.com/crowdstrike/falcon-operator/internal/controller/common"
	pkgcommon "github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/crowdstrike/falcon-operator/pkg/tls"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

// apiTLSValidityDays is the validity of the API mTLS CA and the certificates it signs. The CA and
// its leaves share it so that no leaf outlives the CA that signed it.
const apiTLSValidityDays = 3650

// reconcileAPITLSSecrets ensures the PKI secrets for mTLS between the Cluster Guard Controller
// API service and the node sensor exist. They are only needed while the Cluster Guard Controller
// is enabled, and are removed by its Cleanup when it is disabled.
//
// The CA private key is not kept, so a missing or mismatched secret cannot be re-signed on its own.
// The three secrets are reused together or all regenerated from a new CA.
func (r *FalconClusterGuardReconciler) reconcileAPITLSSecrets(
	ctx context.Context,
	req ctrl.Request,
	fcg *falconv1alpha1.FalconClusterGuard,
) error {
	namespace := fcg.Spec.InstallNamespace

	existing := map[string]*corev1.Secret{}
	for _, name := range []string{
		pkgcommon.ClusterGuardControllerAPITLSSecretName,
		pkgcommon.ClusterGuardControllerAPICASecretName,
		pkgcommon.ClusterGuardNodeSensorTLSSecretName,
	} {
		s := &corev1.Secret{}
		err := pkgcommon.GetNamespacedObject(ctx, r.Client, r.Reader, types.NamespacedName{Name: name, Namespace: namespace}, s)
		if err == nil {
			existing[name] = s
		} else if !apierrors.IsNotFound(err) {
			r.log.Error(err, "Failed to get FalconClusterGuard API mTLS Secret", "name", name)
			return err
		}
	}
	if err := apiTLSSecretsConsistent(existing); err == nil {
		return nil
	} else if len(existing) > 0 {
		r.log.Info("Regenerating the FalconClusterGuard API mTLS secrets", "reason", err.Error())
	}

	// Generate a single CA shared by both the server and client certificates so
	// that mTLS between the API service and the node sensor can be established.
	caCommonName := fmt.Sprintf("falcon-sensor-ca-%s", namespace)
	ca, caKey, err := tls.FCGGenerateCA(caCommonName, apiTLSValidityDays)
	if err != nil {
		r.log.Error(err, "Failed to generate FalconClusterGuard API CA")
		return err
	}

	apiSvcName := fmt.Sprintf("%s.%s.svc", pkgcommon.ClusterGuardControllerAPIServiceName, namespace)
	serverCert, serverKey, err := tls.FCGSignCert(ca, caKey, apiTLSValidityDays, tls.FCGCertInfo{
		CommonName: apiSvcName,
		DNSNames: []string{
			apiSvcName,
			fmt.Sprintf("%s.cluster.local", apiSvcName),
			fmt.Sprintf("%s.%s", pkgcommon.ClusterGuardControllerAPIServiceName, namespace),
			pkgcommon.ClusterGuardControllerAPIServiceName,
		},
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	})
	if err != nil {
		r.log.Error(err, "Failed to generate FalconClusterGuard API server TLS certificate")
		return err
	}

	spiffeURI, _ := url.Parse(fmt.Sprintf("spiffe://cluster.local/ns/%s/sa/falcon-sensor", namespace))
	clientCert, clientKey, err := tls.FCGSignCert(ca, caKey, apiTLSValidityDays, tls.FCGCertInfo{
		CommonName:  "falcon-sensor-client",
		URIs:        []*url.URL{spiffeURI},
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	})
	if err != nil {
		r.log.Error(err, "Failed to generate FalconClusterGuard sensor client TLS certificate")
		return err
	}

	// The CA is written first: if a later write fails, the leaves no longer verify against it and
	// the next reconcile regenerates all three again.
	desired := []*corev1.Secret{
		assets.Secret(pkgcommon.ClusterGuardControllerAPICASecretName, namespace, pkgcommon.ClusterGuardControllerComponentName,
			map[string][]byte{"ca.crt": ca}, corev1.SecretTypeOpaque),
		assets.Secret(pkgcommon.ClusterGuardControllerAPITLSSecretName, namespace, pkgcommon.ClusterGuardControllerComponentName,
			map[string][]byte{"tls.crt": serverCert, "tls.key": serverKey}, corev1.SecretTypeTLS),
		assets.Secret(pkgcommon.ClusterGuardNodeSensorTLSSecretName, namespace, pkgcommon.ClusterGuardControllerComponentName,
			map[string][]byte{"tls.crt": clientCert, "tls.key": clientKey}, corev1.SecretTypeTLS),
	}
	status := &fcg.Status
	for _, s := range desired {
		if e, ok := existing[s.Name]; ok {
			e.Data = s.Data
			e.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Secret"))
			if err := k8sutils.Update(r.Client, ctx, req, r.log, fcg, status, e); err != nil {
				return err
			}
			continue
		}
		maps.Copy(s.Labels, pkgcommon.AppLabels(pkgcommon.ClusterGuardComponentName, fcg.Name, pkgcommon.ClusterGuardControllerAppComponent))
		if err := k8sutils.Create(r, r.RuntimeScheme, ctx, req, r.log, fcg, status, s); err != nil {
			return err
		}
	}

	return nil
}

// apiTLSSecretsConsistent returns an error unless all three API mTLS secrets exist with their keys
// and both leaf certificates are signed by the CA.
func apiTLSSecretsConsistent(existing map[string]*corev1.Secret) error {
	caSecret, ok := existing[pkgcommon.ClusterGuardControllerAPICASecretName]
	if !ok || len(caSecret.Data["ca.crt"]) == 0 {
		return fmt.Errorf("%s is missing or has no ca.crt", pkgcommon.ClusterGuardControllerAPICASecretName)
	}
	for _, name := range []string{pkgcommon.ClusterGuardControllerAPITLSSecretName, pkgcommon.ClusterGuardNodeSensorTLSSecretName} {
		s, ok := existing[name]
		if !ok || len(s.Data["tls.crt"]) == 0 || len(s.Data["tls.key"]) == 0 {
			return fmt.Errorf("%s is missing or has no tls.crt or tls.key", name)
		}
		if err := tls.FCGCheckSignedBy(s.Data["tls.crt"], caSecret.Data["ca.crt"]); err != nil {
			return fmt.Errorf("%s is not signed by the CA in %s: %w", name, pkgcommon.ClusterGuardControllerAPICASecretName, err)
		}
	}
	return nil
}
