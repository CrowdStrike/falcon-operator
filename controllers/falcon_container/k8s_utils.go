package falcon

import (
	"context"
	"fmt"
	"strings"

	"github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *FalconContainerReconciler) Create(ctx context.Context, falconContainer *v1alpha1.FalconContainer, obj runtime.Object) error {
	switch t := obj.(type) {
	case client.Object:
		name := t.GetName()
		namespace := t.GetNamespace()
		if namespace == "" {
			namespace = "(Cluster-Wide)"
		}
		gvk := t.GetObjectKind().GroupVersionKind()
		r.Log.Info(fmt.Sprintf("Creating Falcon Container object %s %s in namespace %s", gvk.Kind, name, namespace))
		err := r.Client.Create(ctx, t)
		if err != nil {
			if errors.IsAlreadyExists(err) {
				r.Log.Info(fmt.Sprintf("Falcon Container object %s %s already exists in namespace %s", gvk.Kind, name, namespace))
			} else {
				return fmt.Errorf("failed to create %s %s in namespace %s: %v", gvk.Kind, name, namespace, err)
			}
		}
		falconContainer.Status.SetCondition(&metav1.Condition{
			Type:    fmt.Sprintf("%sReady", strings.ToUpper(gvk.Kind[:1])+gvk.Kind[1:]),
			Status:  metav1.ConditionTrue,
			Reason:  "Created",
			Message: fmt.Sprintf("Successfully created %s %s in %s", gvk.Kind, name, namespace),
		})
		return r.Client.Status().Update(ctx, falconContainer)
	default:
		return fmt.Errorf("Unrecognized kube object type: %T", obj)
	}
}

func (r *FalconContainerReconciler) Update(ctx context.Context, falconContainer *v1alpha1.FalconContainer, obj runtime.Object) error {
	switch t := obj.(type) {
	case client.Object:
		name := t.GetName()
		namespace := t.GetNamespace()
		if namespace == "" {
			namespace = "(Cluster-Wide)"
		}
		gvk := t.GetObjectKind().GroupVersionKind()
		r.Log.Info(fmt.Sprintf("Updating Falcon Container object %s %s in namespace %s", gvk.Kind, name, namespace))
		err := r.Client.Update(ctx, t)
		if err != nil {
			if errors.IsNotFound(err) {
				r.Log.Info(fmt.Sprintf("Falcon Container object %s %s does not exist in namespace %s", gvk.Kind, name, namespace))
			}
			return fmt.Errorf("Cannot update object %s %s in namespace %s: %v", gvk.Kind, name, namespace, err)
		}
		falconContainer.Status.SetCondition(&metav1.Condition{
			Type:    fmt.Sprintf("%sReady", strings.ToUpper(gvk.Kind[:1])+gvk.Kind[1:]),
			Status:  metav1.ConditionTrue,
			Reason:  "Updated",
			Message: fmt.Sprintf("Successfully updated %s %s in %s", gvk.Kind, name, namespace),
		})
		return r.Client.Status().Update(ctx, falconContainer)
	default:
		return fmt.Errorf("Unrecognized kube object type: %T", obj)
	}
}

func (r *FalconContainerReconciler) Delete(ctx context.Context, falconContainer *v1alpha1.FalconContainer, obj runtime.Object) error {
	switch t := obj.(type) {
	case client.Object:
		name := t.GetName()
		namespace := t.GetNamespace()
		if namespace == "" {
			namespace = "(Cluster-Wide)"
		}
		gvk := t.GetObjectKind().GroupVersionKind()
		r.Log.Info(fmt.Sprintf("Deleting Falcon Container object %s %s in namespace %s", gvk.Kind, name, namespace))
		err := r.Client.Delete(ctx, t)
		if err != nil {
			if errors.IsNotFound(err) {
				r.Log.Info(fmt.Sprintf("Falcon Container object %s %s already removed from namespace %s", gvk.Kind, name, namespace))
			}
			return fmt.Errorf("Cannot delete object %s %s in namespace %s: %v", gvk.Kind, name, namespace, err)
		}
		falconContainer.Status.SetCondition(&metav1.Condition{
			Type:    fmt.Sprintf("%sReady", strings.ToUpper(gvk.Kind[:1])+gvk.Kind[1:]),
			Status:  metav1.ConditionFalse,
			Reason:  "Deleted",
			Message: fmt.Sprintf("Successfully deleted %s %s in %s", gvk.Kind, name, namespace),
		})
		return r.Client.Status().Update(ctx, falconContainer)
	default:
		return fmt.Errorf("Unrecognized kube object type: %T", obj)
	}
}
