package assets

import (
	"github.com/crowdstrike/falcon-operator/pkg/common"
	schedulingv1 "k8s.io/api/scheduling/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func PriorityClass(name string, value *int32) *schedulingv1.PriorityClass {
	defaultValue := int32(1000000000)
	labels := common.CRLabels("priorityclass", name, common.FalconKernelSensor)

	if value == nil {
		value = &defaultValue
	}

	return &schedulingv1.PriorityClass{
		TypeMeta: metav1.TypeMeta{
			APIVersion: schedulingv1.SchemeGroupVersion.String(),
			Kind:       "PriorityClass",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: labels,
		},
		Description: "This priority class would be used to deploy CrowdStrike Falcon node sensor",
		Value:       *value,
	}
}
