package assets

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestNamespace tests the Namespace function
func TestNamespace(t *testing.T) {
	namespace := "test"
	want := &corev1.Namespace{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Namespace",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
	}
	got := Namespace(namespace)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Namespace() mismatch (-want +got): %s", diff)
	}
}
