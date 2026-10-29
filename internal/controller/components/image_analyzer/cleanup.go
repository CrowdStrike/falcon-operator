package image_analyzer

import (
	"context"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	k8sutils "github.com/crowdstrike/falcon-operator/internal/controller/common"
	"github.com/crowdstrike/falcon-operator/internal/controller/components"
	pkgcommon "github.com/crowdstrike/falcon-operator/pkg/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Cleanup removes the Image Analyzer resources owned by the FalconClusterGuard when the component is disabled.
// It is a no-op once the resources are gone.
func (ia *ImageAnalyzer) Cleanup(ctx context.Context) error {
	ns := ia.cfg.InstallNamespace
	if err := components.DeleteOwned(ctx, ia.r, ia.cfg.Owner,
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: pkgcommon.FCGImageAnalyzerDeploymentName, Namespace: ns}},
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: pkgcommon.FalconImageAnalyzerAgentService, Namespace: ns}},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: pkgcommon.FCGImageAnalyzerTLSSecretName, Namespace: ns}},
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: pkgcommon.FCGImageAnalyzerConfigMapName, Namespace: ns}},
		&rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: pkgcommon.FCGImageAnalyzerCRBName}},
		&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: pkgcommon.FCGImageAnalyzerServiceAccountName, Namespace: ns}},
	); err != nil {
		return err
	}
	return k8sutils.ConditionsRemove(ia.r, ctx, ia.cfg.Request, ia.r.GetLog(), ia.cfg.Owner, ia.cfg.Status,
		falconv1alpha1.ConditionImageAnalyzerReady)
}
