package assets

import (
	"github.com/crowdstrike/falcon-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ResourceQuota returns a ResourceQuota object for the admission controller
func ResourceQuota(name string, namespace string, component string) *corev1.ResourceQuota {
	labels := common.CRLabels("resourcequota", name, component)

	return &corev1.ResourceQuota{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ResourceQuota",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: corev1.ResourceList{
				corev1.ResourcePods: resource.MustParse("2"),
			},
			ScopeSelector: &corev1.ScopeSelector{
				MatchExpressions: []corev1.ScopedResourceSelectorRequirement{
					{
						Operator:  corev1.ScopeSelectorOpIn,
						ScopeName: corev1.ResourceQuotaScopePriorityClass,
						Values: []string{
							"system-cluster-critical",
						},
					},
				},
			},
		},
	}
}
