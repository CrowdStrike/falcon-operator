package assets

import (
	"testing"

	"github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// TestService tests the Service function
func TestService(t *testing.T) {
	selector := map[string]string{"test": "test"}
	want := &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
			Labels:    common.CRLabels("service", "test", "test"),
		},
		Spec: corev1.ServiceSpec{
			Selector: selector,
			Ports: []corev1.ServicePort{
				{
					Name:       common.FalconServiceHTTPSName,
					Port:       123,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString(common.FalconServiceHTTPSName),
				},
			},
		},
	}

	got := Service("test", "test", "test", selector, 123)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Service() mismatch (-want +got): %s", diff)
	}
}
