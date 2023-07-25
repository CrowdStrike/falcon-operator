package assets

import (
	"testing"

	"github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestResourceQuota tests the ResourceQuota function
func TestResourceQuota(t *testing.T) {
	want := &corev1.ResourceQuota{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ResourceQuota",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
			Labels:    common.CRLabels("resourcequota", "test", "test"),
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
							common.FalconPriorityClassName,
						},
					},
				},
			},
		},
	}

	got := ResourceQuota("test", "test", "test", "2")
	if diff := cmp.Diff(&want, &got); diff != "" {
		t.Errorf("ResourceQuota() mismatch (-want +got): %s", diff)
	}
}
