package image_analyzer

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

func (ia *ImageAnalyzer) reconcileServiceAccount(ctx context.Context) error {
	sa := assets.ServiceAccount(
		pkgcommon.FCGImageAnalyzerServiceAccountName,
		ia.cfg.InstallNamespace,
		pkgcommon.FCGImageAnalyzerComponentName,
		ia.cfg.ImageAnalyzerSpec.ServiceAccount.Annotations,
		ia.cfg.ImagePullSecrets,
	)
	existing := &corev1.ServiceAccount{}
	found, err := k8sutils.GetOrCreate(ctx, ia.r, ia.cfg.Request, ia.cfg.Owner, ia.cfg.Status, sa, existing,
		types.NamespacedName{Name: pkgcommon.FCGImageAnalyzerServiceAccountName, Namespace: ia.cfg.InstallNamespace},
		"Failed to get FCG Image Analyzer ServiceAccount")
	if !found || err != nil {
		return err
	}

	updated := false
	for k, v := range sa.Annotations {
		if existing.Annotations[k] != v {
			ia.r.GetLog().V(1).Info("Updating FCG Image Analyzer ServiceAccount: annotations changed")
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
			ia.r.GetLog().V(1).Info("Updating FCG Image Analyzer ServiceAccount: labels changed")
			if existing.Labels == nil {
				existing.Labels = make(map[string]string)
			}
			maps.Copy(existing.Labels, sa.Labels)
			updated = true
			break
		}
	}
	if ia.cfg.OpenShift {
		sa.ImagePullSecrets = k8sutils.PreserveOpenShiftPullSecrets(sa.ImagePullSecrets, existing.ImagePullSecrets)
	}
	if !reflect.DeepEqual(sa.ImagePullSecrets, existing.ImagePullSecrets) {
		ia.r.GetLog().V(1).Info("Updating FCG Image Analyzer ServiceAccount: ImagePullSecrets changed",
			"old", existing.ImagePullSecrets,
			"new", sa.ImagePullSecrets)
		existing.ImagePullSecrets = sa.ImagePullSecrets
		updated = true
	}
	if updated {
		existing.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ServiceAccount"))
		return k8sutils.Update(ia.r, ctx, ia.cfg.Request, ia.r.GetLog(), ia.cfg.Owner, ia.cfg.Status, existing)
	}
	return nil
}

func (ia *ImageAnalyzer) reconcileClusterRoleBinding(ctx context.Context) error {
	crb := assets.ClusterRoleBinding(
		pkgcommon.FCGImageAnalyzerCRBName,
		ia.cfg.InstallNamespace,
		pkgcommon.FCGImageAnalyzerClusterRoleName,
		pkgcommon.FCGImageAnalyzerServiceAccountName,
		pkgcommon.FCGImageAnalyzerComponentName,
		[]rbacv1.Subject{},
	)
	existing := &rbacv1.ClusterRoleBinding{}
	found, err := k8sutils.GetOrCreate(ctx, ia.r, ia.cfg.Request, ia.cfg.Owner, ia.cfg.Status, crb, existing,
		types.NamespacedName{Name: pkgcommon.FCGImageAnalyzerCRBName},
		"Failed to get FCG Image Analyzer ClusterRoleBinding")
	if !found || err != nil {
		return err
	}
	if !reflect.DeepEqual(crb.RoleRef, existing.RoleRef) {
		if err := k8sutils.Delete(ia.r, ctx, ia.cfg.Request, ia.r.GetLog(), ia.cfg.Owner, ia.cfg.Status, existing); err != nil {
			return err
		}
		return k8sutils.Create(ia.r, ia.r.GetScheme(), ctx, ia.cfg.Request, ia.r.GetLog(), ia.cfg.Owner, ia.cfg.Status, crb)
	} else if !reflect.DeepEqual(crb.Subjects, existing.Subjects) {
		existing.Subjects = crb.Subjects
		existing.SetGroupVersionKind(rbacv1.SchemeGroupVersion.WithKind("ClusterRoleBinding"))
		return k8sutils.Update(ia.r, ctx, ia.cfg.Request, ia.r.GetLog(), ia.cfg.Owner, ia.cfg.Status, existing)
	}
	return nil
}
