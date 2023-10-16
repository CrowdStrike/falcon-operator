package assets

import (
	"testing"

	"github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/google/go-cmp/cmp"
	imagev1 "github.com/openshift/api/image/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestImageStream tests the OpenShift ImageStream function
func TestImageStream(t *testing.T) {
	want := &imagev1.ImageStream{
		TypeMeta: metav1.TypeMeta{
			APIVersion: imagev1.SchemeGroupVersion.String(),
			Kind:       "ImageStream",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "test",
			Labels:    common.CRLabels("imagestream", "test", "test"),
		},
		Spec: imagev1.ImageStreamSpec{},
	}

	got := ImageStream("test", "test", "test")
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("ImageStream() mismatch (-want +got): %s", diff)
	}
}
