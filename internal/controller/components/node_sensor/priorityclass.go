package node_sensor

import (
	"context"

	"github.com/crowdstrike/falcon-operator/internal/controller/assets"
	k8sutils "github.com/crowdstrike/falcon-operator/internal/controller/common"
	pkgcommon "github.com/crowdstrike/falcon-operator/pkg/common"
	schedulingv1 "k8s.io/api/scheduling/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
)

// reconcilePriorityClass creates or updates a PriorityClass when Deploy is true.
// If a name is provided but Deploy is false or nil, no PriorityClass is created —
// the caller is expected to reference an existing cluster-level resource by name.
// When no name is specified the default is a fixed cluster-wide constant — see rbac.go serviceAccount() for rationale.
func (n *NodeSensor) reconcilePriorityClass(ctx context.Context) error {
	pc := n.cfg.NodeSensor.PriorityClass
	if pc.Deploy == nil || !*pc.Deploy {
		return nil
	}

	name := pc.Name
	if name == "" {
		name = pkgcommon.ClusterGuardNodeSensorPriorityClassName
	}

	desired := assets.PriorityClass(name, pc.Value)

	existing := &schedulingv1.PriorityClass{}
	err := pkgcommon.GetNamespacedObject(ctx, n.r, n.r.GetK8sReader(),
		types.NamespacedName{Name: name}, existing)
	if err != nil && apierrors.IsNotFound(err) {
		n.r.GetLog().Info("Creating FalconClusterGuard PriorityClass", "name", name)
		return k8sutils.Create(n.r, n.r.GetScheme(), ctx, n.cfg.Request, n.r.GetLog(), n.cfg.Owner, n.cfg.Status, desired)
	}
	if err != nil {
		return err
	}

	if pc.Value != nil && existing.Value != *pc.Value {
		n.r.GetLog().Info("Recreating FalconClusterGuard PriorityClass: value changed", "name", name)
		if err := k8sutils.Delete(n.r, ctx, n.cfg.Request, n.r.GetLog(), n.cfg.Owner, n.cfg.Status, existing); err != nil {
			return err
		}
		return k8sutils.Create(n.r, n.r.GetScheme(), ctx, n.cfg.Request, n.r.GetLog(), n.cfg.Owner, n.cfg.Status, desired)
	}

	if existing.Description != desired.Description {
		err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
			if err := pkgcommon.GetNamespacedObject(ctx, n.r, n.r.GetK8sReader(),
				types.NamespacedName{Name: name}, existing); err != nil {
				return err
			}
			existing.Description = desired.Description
			existing.SetGroupVersionKind(schedulingv1.SchemeGroupVersion.WithKind("PriorityClass"))
			return k8sutils.Update(n.r, ctx, n.cfg.Request, n.r.GetLog(), n.cfg.Owner, n.cfg.Status, existing)
		})
		return err
	}

	return nil
}
