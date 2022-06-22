package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeclock "k8s.io/utils/clock"
)

// clock is used to set status condition timestamps.
// This variable makes it easier to test conditions.
var clock kubeclock.Clock = &kubeclock.RealClock{}

// SetCondition adds (or updates) the set of conditions with the given
// condition. It returns a boolean value indicating whether the set condition
// is new or was a change to the existing condition with the same type.
func (status *FalconContainerStatus) SetCondition(newCond *metav1.Condition) bool {
	newCond.LastTransitionTime = metav1.Time{Time: clock.Now()}

	for i, condition := range status.Conditions {
		if condition.Type == newCond.Type {
			if condition.Status == newCond.Status {
				newCond.LastTransitionTime = condition.LastTransitionTime
			}
			changed := condition.Status != newCond.Status ||
				condition.Reason != newCond.Reason ||
				condition.Message != newCond.Message
			status.Conditions[i] = *newCond
			return changed
		}
	}
	status.Conditions = append(status.Conditions, *newCond)
	return true
}

// GetCondition searches the set of conditions for the condition with the given
// ConditionType and returns it. If the matching condition is not found,
// GetCondition returns nil.
func (status *FalconContainerStatus) GetCondition(typ string) *metav1.Condition {
	for i, condition := range status.Conditions {
		if condition.Type == typ {
			return &status.Conditions[i]
		}
	}
	return nil
}

func (status *FalconContainerStatus) SetInitialConditions() {
	conditionTypes := []string{
		"ImageReady",
		"InstallerComplete",
		"Complete",
	}
	for _, typ := range conditionTypes {
		if status.GetCondition(typ) == nil {
			status.SetCondition(&metav1.Condition{
				Type:   typ,
				Status: metav1.ConditionUnknown,
				Reason: "Starting",
			})
		}
	}
}
