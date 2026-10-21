package controllers

import (
	"testing"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestComponentsReadiness(t *testing.T) {
	cond := func(condType string, status metav1.ConditionStatus, reason string) metav1.Condition {
		return metav1.Condition{Type: condType, Status: status, Reason: reason}
	}
	admission := falconv1alpha1.ConditionAdmissionReady
	nodeSensor := falconv1alpha1.ConditionNodeSensorReady
	enabled := []string{admission, nodeSensor}

	tests := []struct {
		name        string
		conditions  []metav1.Condition
		enabled     []string
		wantReady   bool
		wantReason  string
		wantMessage string
	}{
		{
			name: "all enabled components ready",
			conditions: []metav1.Condition{
				cond(admission, metav1.ConditionTrue, falconv1alpha1.ReasonInstallSucceeded),
				cond(nodeSensor, metav1.ConditionTrue, falconv1alpha1.ReasonInstallSucceeded),
			},
			enabled:   enabled,
			wantReady: true,
		},
		{
			name:       "disabled component is ignored",
			conditions: []metav1.Condition{cond(admission, metav1.ConditionTrue, falconv1alpha1.ReasonInstallSucceeded)},
			enabled:    []string{admission},
			wantReady:  true,
		},
		{
			name:        "missing condition counts as progressing",
			conditions:  []metav1.Condition{cond(admission, metav1.ConditionTrue, falconv1alpha1.ReasonInstallSucceeded)},
			enabled:     enabled,
			wantReason:  falconv1alpha1.ReasonProgressing,
			wantMessage: "Waiting for NodeSensorReady",
		},
		{
			name: "any failure makes the result failed",
			conditions: []metav1.Condition{
				cond(admission, metav1.ConditionFalse, falconv1alpha1.ReasonProgressing),
				cond(nodeSensor, metav1.ConditionFalse, falconv1alpha1.ReasonFailed),
			},
			enabled:     enabled,
			wantReason:  falconv1alpha1.ReasonFailed,
			wantMessage: "Waiting for AdmissionReady, NodeSensorReady",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ready, reason, message := componentsReadiness(tt.conditions, tt.enabled)
			if ready != tt.wantReady || reason != tt.wantReason || message != tt.wantMessage {
				t.Errorf("componentsReadiness() = (%v, %q, %q), want (%v, %q, %q)",
					ready, reason, message, tt.wantReady, tt.wantReason, tt.wantMessage)
			}
		})
	}
}
