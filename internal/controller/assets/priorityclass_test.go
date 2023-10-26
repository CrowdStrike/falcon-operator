package assets

import (
	"testing"

	"github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/google/go-cmp/cmp"
	schedulingv1 "k8s.io/api/scheduling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestPriorityClass tests the PriorityClass function
func TestPriorityClass(t *testing.T) {
	name := "test"
	value := int32(1000000000)
	want := &schedulingv1.PriorityClass{
		TypeMeta: metav1.TypeMeta{
			APIVersion: schedulingv1.SchemeGroupVersion.String(),
			Kind:       "PriorityClass",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: common.CRLabels("priorityclass", name, common.FalconKernelSensor),
		},
		Description: "This priority class would be used to deploy CrowdStrike Falcon node sensor",
		Value:       value,
	}

	// Test with nil value
	got := PriorityClass(name, nil)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("PriorityClass() mismatch (-want +got): %s", diff)
	}

	// Test with defined value
	got = PriorityClass(name, &value)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("PriorityClass() mismatch (-want +got): %s", diff)
	}
}
