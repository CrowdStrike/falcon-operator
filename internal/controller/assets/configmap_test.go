package assets

import (
	"testing"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/crowdstrike/falcon-operator/pkg/node"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestDaemonsetConfigMap tests the SensorConfigMap function
func TestSensorConfigMap(t *testing.T) {
	falconNode := falconv1alpha1.FalconNodeSensor{}
	falconNode.Spec.FalconAPI = nil
	falconCID := "1234567890ABCDEF1234567890ABCDEF-12"
	falconImage := "testMyImage"
	cfgName := "test"

	want := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cfgName,
			Namespace: cfgName,
			Labels:    common.CRLabels("configmap", cfgName, common.FalconKernelSensor),
		},
		Data: map[string]string{
			"FALCONCTL_OPT_CID": "1234567890ABCDEF1234567890ABCDEF-12",
		},
	}

	config := node.ConfigCacheTest(falconCID, falconImage, &falconNode, nil)

	got := SensorConfigMap("test", "test", common.FalconKernelSensor, config.SensorEnvVars())
	if diff := cmp.Diff(&want, &got); diff != "" {
		t.Errorf("SensorConfigMap() mismatch (-want +got): %s", diff)
	}
}
