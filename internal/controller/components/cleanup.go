package components

import (
	"context"

	k8sutils "github.com/crowdstrike/falcon-operator/internal/controller/common"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

// GetOwned fetches obj by its name and namespace and reports whether it exists and is controlled by owner.
func GetOwned(ctx context.Context, r k8sutils.Reconciler, owner client.Object, obj client.Object) (bool, error) {
	if err := r.GetK8sReader().Get(ctx, client.ObjectKeyFromObject(obj), obj); err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return metav1.IsControlledBy(obj, owner), nil
}

// DeleteOwned deletes each object that exists and is controlled by owner, in the order given.
// Objects that are missing or owned by something else (e.g. a standalone CR using the same name)
// are skipped, so it is safe to call on every reconcile of a disabled component.
func DeleteOwned(ctx context.Context, r k8sutils.Reconciler, owner client.Object, objs ...client.Object) error {
	log := r.GetLog()
	for _, obj := range objs {
		owned, err := GetOwned(ctx, r, owner, obj)
		if err != nil {
			return err
		}
		if !owned {
			continue
		}

		kind := "object"
		if gvk, err := apiutil.GVKForObject(obj, r.GetScheme()); err == nil {
			kind = gvk.Kind
		}
		log.Info("Deleting FalconClusterGuard component resource",
			"kind", kind, "name", obj.GetName(), "namespace", obj.GetNamespace())
		if err := r.Delete(ctx, obj, client.PropagationPolicy(metav1.DeletePropagationBackground)); err != nil && !apierrors.IsNotFound(err) {
			log.Error(err, "Failed to delete FalconClusterGuard component resource",
				"kind", kind, "name", obj.GetName(), "namespace", obj.GetNamespace())
			return err
		}
	}
	return nil
}
