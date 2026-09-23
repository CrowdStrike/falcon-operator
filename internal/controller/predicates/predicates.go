package predicates

import (
	"reflect"
	"strings"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// CrowdStrikeLabel returns a predicate that skips reconciles triggered solely
// by metadata changes (labels or annotations) where no key contains "crowdstrike.com".
// Create, delete, and generic events are always allowed through.
func CrowdStrikeLabel() predicate.Predicate {
	return predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			// Generation change means spec changed — always reconcile.
			if e.ObjectOld.GetGeneration() != e.ObjectNew.GetGeneration() {
				return true
			}

			// Content beyond metadata changed — always reconcile.
			if !onlyMetadataChanged(e.ObjectOld, e.ObjectNew) {
				return true
			}

			// Metadata-only change: only reconcile if a crowdstrike.com key
			// is involved in labels or annotations.
			return hasCrowdStrikeKey(e.ObjectOld.GetLabels()) ||
				hasCrowdStrikeKey(e.ObjectNew.GetLabels()) ||
				hasCrowdStrikeKey(e.ObjectOld.GetAnnotations()) ||
				hasCrowdStrikeKey(e.ObjectNew.GetAnnotations())
		},
		CreateFunc:  func(event.CreateEvent) bool { return true },
		DeleteFunc:  func(event.DeleteEvent) bool { return true },
		GenericFunc: func(event.GenericEvent) bool { return true },
	}
}

// hasCrowdStrikeKey reports whether any key in m contains "crowdstrike.com".
func hasCrowdStrikeKey(m map[string]string) bool {
	for k := range m {
		if strings.Contains(k, "crowdstrike.com") {
			return true
		}
	}
	return false
}

// onlyMetadataChanged reports whether the two objects differ only in volatile
// or cosmetic metadata fields (labels, annotations, resourceVersion, managedFields).
// Returns false on conversion error, which causes the caller to reconcile.
func onlyMetadataChanged(oldObj, newObj client.Object) bool {
	oldMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(oldObj)
	if err != nil {
		return false
	}
	newMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(newObj)
	if err != nil {
		return false
	}

	for _, m := range []map[string]any{oldMap, newMap} {
		if meta, ok := m["metadata"].(map[string]any); ok {
			delete(meta, "labels")
			delete(meta, "annotations")
			delete(meta, "resourceVersion")
			delete(meta, "managedFields")
		}
	}

	return reflect.DeepEqual(oldMap, newMap)
}
