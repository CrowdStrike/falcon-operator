package clusterguard_controller

import (
	"context"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	k8sutils "github.com/crowdstrike/falcon-operator/internal/controller/common"
	"github.com/crowdstrike/falcon-operator/internal/controller/components"
	pkgcommon "github.com/crowdstrike/falcon-operator/pkg/common"
	arv1 "k8s.io/api/admissionregistration/v1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Cleanup removes the admission controller resources owned by the FalconClusterGuard when the component is disabled.
// The API TLS secrets are left in place because they are shared with the node sensor.
// It is a no-op once the resources are gone.
func (a *ClusterGuardController) Cleanup(ctx context.Context) error {
	ns := a.cfg.InstallNamespace
	if err := components.DeleteOwned(ctx, a.r, a.cfg.Owner,
		// Remove the webhook first so admission requests are not sent to a backend that is going away.
		&arv1.ValidatingWebhookConfiguration{ObjectMeta: metav1.ObjectMeta{Name: pkgcommon.ClusterGuardControllerValidatingWebhookName}},
		&appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: pkgcommon.ClusterGuardControllerDeploymentName, Namespace: ns}},
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: pkgcommon.ClusterGuardControllerWebhookServiceName, Namespace: ns}},
		&corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: pkgcommon.ClusterGuardControllerAPIServiceName, Namespace: ns}},
		&corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: pkgcommon.ClusterGuardControllerTLSSecretName, Namespace: ns}},
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: pkgcommon.ClusterGuardControllerConfigMapName, Namespace: ns}},
		&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: pkgcommon.FalconAdmissionClusterNameConfigMapName, Namespace: ns}},
		&corev1.ResourceQuota{ObjectMeta: metav1.ObjectMeta{Name: pkgcommon.ClusterGuardControllerResourceQuotaName, Namespace: ns}},
		&rbacv1.RoleBinding{ObjectMeta: metav1.ObjectMeta{Name: pkgcommon.ClusterGuardControllerRoleBindingName, Namespace: ns}},
		&rbacv1.Role{ObjectMeta: metav1.ObjectMeta{Name: pkgcommon.ClusterGuardControllerNamespaceRoleName, Namespace: ns}},
		&rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: pkgcommon.ClusterGuardControllerClusterRoleBindingName}},
		&corev1.ServiceAccount{ObjectMeta: metav1.ObjectMeta{Name: pkgcommon.ClusterGuardControllerServiceAccountName, Namespace: ns}},
	); err != nil {
		return err
	}
	return k8sutils.ConditionsRemove(a.r, ctx, a.cfg.Request, a.r.GetLog(), a.cfg.Owner, a.cfg.Status,
		falconv1alpha1.ConditionAdmissionReady)
}
