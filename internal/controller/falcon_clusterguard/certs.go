package controllers

import (
	"context"
	"crypto/x509"
	"fmt"
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

// reconcileAPITLSSecrets ensures the PKI secrets shared between the admission
// controller and node sensor exist. They are managed here rather than per-component
// so they are created regardless of which component is enabled.
func (r *FalconClusterGuardReconciler) reconcileAPITLSSecrets(
	ctx context.Context,
	req ctrl.Request,
	fcg *falconv1alpha1.FalconClusterGuard,
) error {
	namespace := fcg.Spec.InstallNamespace

	existingAPI := &corev1.Secret{}
	errAPI := pkgcommon.GetNamespacedObject(ctx, r.Client, r.Reader, types.NamespacedName{Name: pkgcommon.ClusterGuardControllerAPITLSSecretName, Namespace: namespace}, existingAPI)
	existingCA := &corev1.Secret{}
	errCA := pkgcommon.GetNamespacedObject(ctx, r.Client, r.Reader, types.NamespacedName{Name: pkgcommon.ClusterGuardControllerAPICASecretName, Namespace: namespace}, existingCA)
	existingSensor := &corev1.Secret{}
	errSensor := pkgcommon.GetNamespacedObject(ctx, r.Client, r.Reader, types.NamespacedName{Name: pkgcommon.ClusterGuardNodeSensorTLSSecretName, Namespace: namespace}, existingSensor)
	if errAPI == nil && errCA == nil && errSensor == nil {
		return nil
	}

	// Generate a single CA shared by both the server and client certificates so
	// that mTLS between the API service and the node sensor can be established.
	caCommonName := fmt.Sprintf("falcon-sensor-ca-%s", namespace)
	ca, caKey, err := tls.FCGGenerateCA(caCommonName, 730)
	if err != nil {
		r.log.Error(err, "Failed to generate FalconClusterGuard API CA")
		return err
	}

	apiSvcName := fmt.Sprintf("%s.%s.svc", pkgcommon.ClusterGuardControllerAPIServiceName, namespace)
	serverCert, serverKey, err := tls.FCGSignCert(ca, caKey, 3650, tls.FCGCertInfo{
		CommonName:  apiSvcName,
		DNSNames:    []string{apiSvcName, fmt.Sprintf("%s.cluster.local", apiSvcName)},
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	})
	if err != nil {
		r.log.Error(err, "Failed to generate FalconClusterGuard API server TLS certificate")
		return err
	}

	spiffeURI, _ := url.Parse(fmt.Sprintf("spiffe://cluster.local/ns/%s/sa/falcon-sensor", namespace))
	clientCert, clientKey, err := tls.FCGSignCert(ca, caKey, 3650, tls.FCGCertInfo{
		CommonName:  "falcon-sensor-client",
		URIs:        []*url.URL{spiffeURI},
		ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth},
	})
	if err != nil {
		r.log.Error(err, "Failed to generate FalconClusterGuard sensor client TLS certificate")
		return err
	}

	status := &fcg.Status
	if apierrors.IsNotFound(errAPI) {
		s := assets.Secret(pkgcommon.ClusterGuardControllerAPITLSSecretName, namespace, pkgcommon.ClusterGuardControllerComponentName,
			map[string][]byte{"tls.crt": serverCert, "tls.key": serverKey}, corev1.SecretTypeTLS)
		if err := k8sutils.Create(r, r.RuntimeScheme, ctx, req, r.log, fcg, status, s); err != nil {
			return err
		}
	} else if errAPI != nil {
		r.log.Error(errAPI, "Failed to get FalconClusterGuard API TLS Secret")
		return errAPI
	}

	if apierrors.IsNotFound(errCA) {
		s := assets.Secret(pkgcommon.ClusterGuardControllerAPICASecretName, namespace, pkgcommon.ClusterGuardControllerComponentName,
			map[string][]byte{"ca.crt": ca}, corev1.SecretTypeOpaque)
		if err := k8sutils.Create(r, r.RuntimeScheme, ctx, req, r.log, fcg, status, s); err != nil {
			return err
		}
	} else if errCA != nil {
		r.log.Error(errCA, "Failed to get FalconClusterGuard API CA Secret")
		return errCA
	}

	if apierrors.IsNotFound(errSensor) {
		s := assets.Secret(pkgcommon.ClusterGuardNodeSensorTLSSecretName, namespace, pkgcommon.ClusterGuardControllerComponentName,
			map[string][]byte{"tls.crt": clientCert, "tls.key": clientKey}, corev1.SecretTypeTLS)
		if err := k8sutils.Create(r, r.RuntimeScheme, ctx, req, r.log, fcg, status, s); err != nil {
			return err
		}
	} else if errSensor != nil {
		r.log.Error(errSensor, "Failed to get FalconClusterGuard sensor TLS Secret")
		return errSensor
	}

	return nil
}
