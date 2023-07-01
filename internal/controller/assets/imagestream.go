package assets

import (
	imagev1 "github.com/openshift/api/image/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ImageStream returns an OpenShift ImageStream object
func ImageStream(name string, namespace string) *imagev1.ImageStream {
	return &imagev1.ImageStream{
		TypeMeta: metav1.TypeMeta{
			APIVersion: imagev1.SchemeGroupVersion.String(),
			Kind:       "ImageStream",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: imagev1.ImageStreamSpec{},
	}
}
