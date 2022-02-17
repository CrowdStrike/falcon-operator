package k8s_utils

import (
	"k8s.io/apimachinery/pkg/runtime"
)

func PopNamespaceFromObjectList(objects []runtime.Object) (nsObject runtime.Object, otherObjects []runtime.Object) {
	otherObjects = []runtime.Object{}
	for i := range objects {
		object := &objects[i]
		if isNamespaceObject(*object) {
			nsObject = *object
		} else {
			otherObjects = append(otherObjects, *object)
		}
	}
	return
}

func isNamespaceObject(obj runtime.Object) bool {
	versionKind := obj.GetObjectKind().GroupVersionKind()
	return versionKind.Group == "" && versionKind.Kind == "Namespace"
}
