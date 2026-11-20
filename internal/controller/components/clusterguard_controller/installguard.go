package clusterguard_controller

import (
	"context"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	k8sutils "github.com/crowdstrike/falcon-operator/internal/controller/common"
	pkgcommon "github.com/crowdstrike/falcon-operator/pkg/common"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

// installGuardOwner builds the InstallGuardOwner that the clusterguard controller sets as the owner of the install
// guard ValidatingWebhookConfiguration it creates while uninstall protection is enabled, so the VWC is
// garbage-collected with the FalconClusterGuard. The controller looks it up by its fixed name.
func (a *ClusterGuardController) installGuardOwner() *falconv1alpha1.InstallGuardOwner {
	return &falconv1alpha1.InstallGuardOwner{
		TypeMeta: metav1.TypeMeta{
			APIVersion: falconv1alpha1.GroupVersion.String(),
			Kind:       "InstallGuardOwner",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: pkgcommon.ClusterGuardInstallGuardOwnerName,
			Labels: a.withAppLabels(pkgcommon.CRLabels("installguardowner", pkgcommon.ClusterGuardInstallGuardOwnerName,
				pkgcommon.ClusterGuardControllerComponentName)),
		},
	}
}

// reconcileInstallGuardOwner creates the InstallGuardOwner. It holds no data, so an existing one is left as is.
func (a *ClusterGuardController) reconcileInstallGuardOwner(ctx context.Context) error {
	_, err := k8sutils.GetOrCreate(ctx, a.r, a.cfg.Request, a.cfg.Owner, a.cfg.Status, a.installGuardOwner(),
		&falconv1alpha1.InstallGuardOwner{}, types.NamespacedName{Name: pkgcommon.ClusterGuardInstallGuardOwnerName},
		"Failed to get FalconClusterGuard InstallGuardOwner")
	return err
}
