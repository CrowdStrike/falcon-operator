package admission

import (
	"context"
	"maps"
	"reflect"

	"github.com/crowdstrike/falcon-operator/internal/controller/assets"
	k8sutils "github.com/crowdstrike/falcon-operator/internal/controller/common"
	pkgcommon "github.com/crowdstrike/falcon-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/types"
)

// serviceAccount builds the ServiceAccount for the FalconClusterGuard admission controller.
func (a *Admission) serviceAccount() *corev1.ServiceAccount {
	return assets.ServiceAccount(a.prefix()+"-sa", a.cfg.InstallNamespace, pkgcommon.AdmissionComponentName, a.cfg.AdmissionConfig.ServiceAccount.Annotations, a.cfg.ImagePullSecrets)
}

// clusterRoleBinding builds the ClusterRoleBinding for the FalconClusterGuard admission controller.
func (a *Admission) clusterRoleBinding() *rbacv1.ClusterRoleBinding {
	return assets.ClusterRoleBinding(
		a.prefix()+"-security-crb",
		a.cfg.InstallNamespace,
		pkgcommon.FCGAdmissionClusterRoleName,
		a.prefix()+"-sa",
		pkgcommon.AdmissionComponentName,
		[]rbacv1.Subject{},
	)
}

// roleBinding builds the RoleBinding for the FalconClusterGuard admission controller.
func (a *Admission) roleBinding() *rbacv1.RoleBinding {
	return assets.RoleBinding(a.prefix()+"-rolebinding", a.cfg.InstallNamespace, a.prefix()+"-namespace-role", a.prefix()+"-sa")
}

// role builds the Role for the FalconClusterGuard admission controller.
func (a *Admission) role() *rbacv1.Role {
	return assets.Role(a.prefix()+"-namespace-role", a.cfg.InstallNamespace)
}

// reconcileServiceAccount reconciles the ServiceAccount for the admission controller.
func (a *Admission) reconcileServiceAccount(ctx context.Context) error {
	sa := a.serviceAccount()
	existing := &corev1.ServiceAccount{}
	found, err := k8sutils.GetOrCreate(ctx, a.r, a.cfg.Request, a.cfg.Owner, a.cfg.Status, sa, existing,
		types.NamespacedName{Name: a.prefix() + "-sa", Namespace: a.cfg.InstallNamespace},
		"Failed to get FalconClusterGuard ServiceAccount")
	if !found || err != nil {
		return err
	}

	updated := false

	for k, v := range sa.Annotations {
		if existing.Annotations[k] != v {
			a.r.GetLog().V(1).Info("Updating FalconClusterGuard ServiceAccount: annotations changed")
			if existing.Annotations == nil {
				existing.Annotations = make(map[string]string)
			}
			maps.Copy(existing.Annotations, sa.Annotations)
			updated = true
			break
		}
	}

	for k, v := range sa.Labels {
		if existing.Labels[k] != v {
			a.r.GetLog().V(1).Info("Updating FalconClusterGuard ServiceAccount: labels changed")
			if existing.Labels == nil {
				existing.Labels = make(map[string]string)
			}
			maps.Copy(existing.Labels, sa.Labels)
			updated = true
			break
		}
	}

	if !reflect.DeepEqual(sa.ImagePullSecrets, existing.ImagePullSecrets) {
		a.r.GetLog().V(1).Info("Updating FalconClusterGuard ServiceAccount: ImagePullSecrets changed",
			"old", existing.ImagePullSecrets,
			"new", sa.ImagePullSecrets)
		existing.ImagePullSecrets = sa.ImagePullSecrets
		updated = true
	}

	if updated {
		existing.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ServiceAccount"))
		return k8sutils.Update(a.r, ctx, a.cfg.Request, a.r.GetLog(), a.cfg.Owner, a.cfg.Status, existing)
	}
	return nil
}

func (a *Admission) reconcileClusterRoleBinding(ctx context.Context) error {
	crb := a.clusterRoleBinding()
	existing := &rbacv1.ClusterRoleBinding{}
	found, err := k8sutils.GetOrCreate(ctx, a.r, a.cfg.Request, a.cfg.Owner, a.cfg.Status, crb, existing,
		types.NamespacedName{Name: a.prefix() + "-security-crb"},
		"Failed to get FalconClusterGuard ClusterRoleBinding")
	if !found || err != nil {
		return err
	}
	if !reflect.DeepEqual(crb.RoleRef, existing.RoleRef) {
		a.r.GetLog().V(1).Info("Recreating FalconClusterGuard ClusterRoleBinding: RoleRef changed")
		if err := k8sutils.Delete(a.r, ctx, a.cfg.Request, a.r.GetLog(), a.cfg.Owner, a.cfg.Status, existing); err != nil {
			return err
		}
		return k8sutils.Create(a.r, a.r.GetScheme(), ctx, a.cfg.Request, a.r.GetLog(), a.cfg.Owner, a.cfg.Status, crb)
	} else if !reflect.DeepEqual(crb.Subjects, existing.Subjects) {
		a.r.GetLog().V(1).Info("Updating FalconClusterGuard ClusterRoleBinding: subjects changed")
		existing.Subjects = crb.Subjects
		existing.SetGroupVersionKind(rbacv1.SchemeGroupVersion.WithKind("ClusterRoleBinding"))
		return k8sutils.Update(a.r, ctx, a.cfg.Request, a.r.GetLog(), a.cfg.Owner, a.cfg.Status, existing)
	}
	return nil
}

func (a *Admission) reconcileRoleBinding(ctx context.Context) error {
	rb := a.roleBinding()
	existing := &rbacv1.RoleBinding{}
	found, err := k8sutils.GetOrCreate(ctx, a.r, a.cfg.Request, a.cfg.Owner, a.cfg.Status, rb, existing,
		types.NamespacedName{Name: a.prefix() + "-rolebinding", Namespace: a.cfg.InstallNamespace},
		"Failed to get FalconClusterGuard RoleBinding")
	if !found || err != nil {
		return err
	}
	if !reflect.DeepEqual(rb.RoleRef, existing.RoleRef) {
		a.r.GetLog().V(1).Info("Recreating FalconClusterGuard RoleBinding: RoleRef changed")
		if err := k8sutils.Delete(a.r, ctx, a.cfg.Request, a.r.GetLog(), a.cfg.Owner, a.cfg.Status, existing); err != nil {
			return err
		}
		return k8sutils.Create(a.r, a.r.GetScheme(), ctx, a.cfg.Request, a.r.GetLog(), a.cfg.Owner, a.cfg.Status, rb)
	} else if !reflect.DeepEqual(rb.Subjects, existing.Subjects) {
		a.r.GetLog().V(1).Info("Updating FalconClusterGuard RoleBinding: subjects changed")
		existing.Subjects = rb.Subjects
		existing.SetGroupVersionKind(rbacv1.SchemeGroupVersion.WithKind("RoleBinding"))
		return k8sutils.Update(a.r, ctx, a.cfg.Request, a.r.GetLog(), a.cfg.Owner, a.cfg.Status, existing)
	}
	return nil
}

func (a *Admission) reconcileRole(ctx context.Context) error {
	role := a.role()
	existing := &rbacv1.Role{}
	found, err := k8sutils.GetOrCreate(ctx, a.r, a.cfg.Request, a.cfg.Owner, a.cfg.Status, role, existing,
		types.NamespacedName{Name: a.prefix() + "-namespace-role", Namespace: a.cfg.InstallNamespace},
		"Failed to get FalconClusterGuard Role")
	if !found || err != nil {
		return err
	}
	if !reflect.DeepEqual(role.Rules, existing.Rules) {
		a.r.GetLog().V(1).Info("Updating FalconClusterGuard Role: rules changed")
		existing.Rules = role.Rules
		existing.SetGroupVersionKind(rbacv1.SchemeGroupVersion.WithKind("Role"))
		return k8sutils.Update(a.r, ctx, a.cfg.Request, a.r.GetLog(), a.cfg.Owner, a.cfg.Status, existing)
	}
	return nil
}
