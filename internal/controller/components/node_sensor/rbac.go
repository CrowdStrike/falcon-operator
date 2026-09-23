package node_sensor

import (
	"context"
	"reflect"

	"github.com/crowdstrike/falcon-operator/internal/controller/assets"
	k8sutils "github.com/crowdstrike/falcon-operator/internal/controller/common"
	pkgcommon "github.com/crowdstrike/falcon-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/types"
)

// serviceAccount builds the ServiceAccount for the node sensor.
func (n *NodeSensor) serviceAccount() *corev1.ServiceAccount {
	return assets.ServiceAccount(pkgcommon.ClusterGuardSensorServiceAccountName, n.cfg.InstallNamespace, pkgcommon.ClusterGuardComponentName, n.cfg.NodeSensor.ServiceAccount.Annotations, n.cfg.ImagePullSecrets)
}

// clusterRoleBinding builds the ClusterRoleBinding for the node sensor.
func (n *NodeSensor) clusterRoleBinding() *rbacv1.ClusterRoleBinding {
	return assets.ClusterRoleBinding(
		n.prefix()+"-sensor-crb",
		n.cfg.InstallNamespace,
		pkgcommon.ClusterGuardSensorClusterRoleName,
		pkgcommon.ClusterGuardSensorServiceAccountName,
		pkgcommon.ClusterGuardComponentName,
		[]rbacv1.Subject{},
	)
}

// cleanupServiceAccount builds the ServiceAccount for the node sensor cleanup DaemonSet.
func (n *NodeSensor) cleanupServiceAccount() *corev1.ServiceAccount {
	return assets.ServiceAccount(pkgcommon.ClusterGuardSensorCleanupServiceAccountName, n.cfg.InstallNamespace, pkgcommon.ClusterGuardComponentName, nil, nil)
}

func (n *NodeSensor) reconcileServiceAccount(ctx context.Context) error {
	sa := n.serviceAccount()
	_, err := k8sutils.GetOrCreate(ctx, n.r, n.cfg.Request, n.cfg.Owner, n.cfg.Status, sa, &corev1.ServiceAccount{},
		types.NamespacedName{Name: pkgcommon.ClusterGuardSensorServiceAccountName, Namespace: n.cfg.InstallNamespace},
		"Failed to get FalconClusterGuard sensor ServiceAccount")
	return err
}

func (n *NodeSensor) reconcileClusterRoleBinding(ctx context.Context) error {
	crb := n.clusterRoleBinding()
	existing := &rbacv1.ClusterRoleBinding{}
	found, err := k8sutils.GetOrCreate(ctx, n.r, n.cfg.Request, n.cfg.Owner, n.cfg.Status, crb, existing,
		types.NamespacedName{Name: n.prefix() + "-sensor-crb"},
		"Failed to get FalconClusterGuard sensor ClusterRoleBinding")
	if !found || err != nil {
		return err
	}
	if !reflect.DeepEqual(crb.RoleRef, existing.RoleRef) {
		n.r.GetLog().V(1).Info("Recreating FalconClusterGuard sensor ClusterRoleBinding: RoleRef changed")
		if err := k8sutils.Delete(n.r, ctx, n.cfg.Request, n.r.GetLog(), n.cfg.Owner, n.cfg.Status, existing); err != nil {
			return err
		}
		return k8sutils.Create(n.r, n.r.GetScheme(), ctx, n.cfg.Request, n.r.GetLog(), n.cfg.Owner, n.cfg.Status, crb)
	} else if !reflect.DeepEqual(crb.Subjects, existing.Subjects) {
		n.r.GetLog().V(1).Info("Updating FalconClusterGuard sensor ClusterRoleBinding: subjects changed")
		existing.Subjects = crb.Subjects
		existing.SetGroupVersionKind(rbacv1.SchemeGroupVersion.WithKind("ClusterRoleBinding"))
		return k8sutils.Update(n.r, ctx, n.cfg.Request, n.r.GetLog(), n.cfg.Owner, n.cfg.Status, existing)
	}
	return nil
}
