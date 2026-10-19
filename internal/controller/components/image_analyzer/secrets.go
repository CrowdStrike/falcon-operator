package image_analyzer

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

func (ia *ImageAnalyzer) reconcileTLSSecret(ctx context.Context) error {
	existing := &corev1.Secret{}
	err := pkgcommon.GetNamespacedObject(ctx, ia.r, ia.r.GetK8sReader(),
		types.NamespacedName{Name: pkgcommon.FCGImageAnalyzerTLSSecretName, Namespace: ia.cfg.InstallNamespace}, existing)
	if err == nil {
		// Secret already exists; cert rotation is not handled here.
		return nil
	}
	if !apierrors.IsNotFound(err) {
		ia.r.GetLog().Error(err, "Failed to get FCG Image Analyzer TLS Secret")
		return err
	}

	spec := ia.cfg.ImageAnalyzerSpec
	namespace := ia.cfg.InstallNamespace
	validity := spec.IARAgentService.CertExpiration
	domainName := spec.IARAgentService.DomainName

	fullName := fmt.Sprintf("%s.%s.svc", pkgcommon.FalconImageAnalyzerAgentService, namespace)
	if domainName != "" {
		fullName = fmt.Sprintf("%s.%s.svc.%s", pkgcommon.FalconImageAnalyzerAgentService, namespace, domainName)
	}

	altDNSNames := []string{
		fullName,
		fmt.Sprintf("%s.cluster.local", fullName),
		fmt.Sprintf("%s.%s", fullName, namespace),
	}

	certInfo := tls.CertInfo{
		CommonName: fullName,
		DNSNames:   altDNSNames,
	}

	c, k, b, err := tls.CertSetup(namespace, validity, certInfo)
	if err != nil {
		ia.r.GetLog().Error(err, "Failed to generate FCG Image Analyzer TLS certificates")
		return err
	}

	secretData := map[string][]byte{
		"tls.crt": c,
		"tls.key": k,
		"ca.crt":  b,
	}

	// Labels required for KAC -> IAR communication; values must match the standalone FalconImageAnalyzer.
	labels := pkgcommon.CRLabels("secret", pkgcommon.FCGImageAnalyzerTLSSecretName, pkgcommon.FCGImageAnalyzerComponentName)
	labels[pkgcommon.AppLabelKey] = pkgcommon.FalconImageAnalyzerAgentServiceApp
	labels[pkgcommon.KubernetesComponentKey] = pkgcommon.FalconImageAnalyzerComponentName
	labels[pkgcommon.KubernetesNameKey] = pkgcommon.FCGImageAnalyzerDeploymentName

	secret := assets.SecretWithCustomLabels(pkgcommon.FCGImageAnalyzerTLSSecretName, namespace, secretData, corev1.SecretTypeTLS, labels)
	return k8sutils.Create(ia.r, ia.r.GetScheme(), ctx, ia.cfg.Request, ia.r.GetLog(), ia.cfg.Owner, ia.cfg.Status, secret)
}
