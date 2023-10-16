package assets

import (
	"github.com/crowdstrike/falcon-operator/pkg/common"
	imagev1 "github.com/openshift/api/image/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ImageStream returns an OpenShift ImageStream object
func ImageStream(name string, namespace string, component string) *imagev1.ImageStream {
	labels := common.CRLabels("imagestream", name, component)

	return &imagev1.ImageStream{
		TypeMeta: metav1.TypeMeta{
			APIVersion: imagev1.SchemeGroupVersion.String(),
			Kind:       "ImageStream",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: imagev1.ImageStreamSpec{},
	}
}
