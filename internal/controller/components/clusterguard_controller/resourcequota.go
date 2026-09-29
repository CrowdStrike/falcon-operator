package clusterguard_controller

import (
	"context"
	"strconv"

	"github.com/crowdstrike/falcon-operator/internal/controller/assets"
	k8sutils "github.com/crowdstrike/falcon-operator/internal/controller/common"
	pkgcommon "github.com/crowdstrike/falcon-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	apiequality "k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const defaultResourceQuotaPods = int32(2)

func (a *ClusterGuardController) getResourceQuotaPods() int32 {
	if a.cfg.ClusterGuardControllerConfig.ResourceQuotaPods == nil {
		return defaultResourceQuotaPods
	}
	return *a.cfg.ClusterGuardControllerConfig.ResourceQuotaPods
}

func (a *ClusterGuardController) resourceQuota() *corev1.ResourceQuota {
	pods := strconv.Itoa(int(a.getResourceQuotaPods()))
	return assets.ResourceQuota(a.prefix()+"-quota", a.cfg.InstallNamespace, pkgcommon.AdmissionComponentName, pods)
}

func (a *ClusterGuardController) reconcileResourceQuota(ctx context.Context) error {
	rq := a.resourceQuota()
	existing := &corev1.ResourceQuota{}
	found, err := k8sutils.GetOrCreate(ctx, a.r, a.cfg.Request, a.cfg.Owner, a.cfg.Status, rq, existing,
		types.NamespacedName{Name: a.prefix() + "-quota", Namespace: a.cfg.InstallNamespace},
		"Failed to get FalconClusterGuard ResourceQuota")
	if !found || err != nil {
		return err
	}
	if !apiequality.Semantic.DeepEqual(rq.Spec, existing.Spec) {
		a.r.GetLog().V(1).Info("Patching FalconClusterGuard ResourceQuota: spec changed")
		base := existing.DeepCopyObject().(client.Object)
		existing.Spec = rq.Spec
		existing.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ResourceQuota"))
		return k8sutils.Patch(a.r, ctx, a.cfg.Request, a.r.GetLog(), a.cfg.Owner, a.cfg.Status, existing, client.MergeFrom(base))
	}
	return nil
}
