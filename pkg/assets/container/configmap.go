package container

import (
	"encoding/json"
	"fmt"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ContainerConfigMap(dsname string, falconContainer *falconv1alpha1.FalconContainer) *corev1.ConfigMap {
	containerData := make(map[string]string)
	containerData["CP_NAMESPACE"] = falconContainer.Namespace
	containerData["FALCON_IMAGE"] = falconContainer.Spec.FalconContainerSensorConfig.Image
	containerData["FALCON_IMAGE_PULL_POLICY"] = string(falconContainer.Spec.FalconContainerSensorConfig.ImagePullPolicy)
	containerData["FALCON_INJECTOR_LISTEN_PORT"] = fmt.Sprintf("%d", falconContainer.Spec.FalconContainerSensorConfig.InjectorPort)

	resources, _ := json.Marshal(falconContainer.Spec.FalconContainerSensor.ContainerResources)
	if string(resources) != "null" {
		containerData["FALCON_RESOURCES"] = string(common.EncodedBase64String(string(resources)))
	}
	if falconContainer.Spec.FalconContainerSensorConfig.DisablePodInjection {
		containerData["INJECTION_DEFAULT_DISABLED"] = "T"
	}
	if len(falconContainer.Spec.FalconContainerSensorConfig.ContainerDaemonSocket) > 0 {
		containerData["SENSOR_CTR_RUNTIME_SOCKET_PATH"] = falconContainer.Spec.FalconContainerSensorConfig.ContainerDaemonSocket
	}

	for k, v := range common.FalconContainerConfig(falconContainer) {
		containerData[k] = v
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dsname,
			Namespace: falconContainer.Namespace,
			Labels: map[string]string{
				common.FalconInstanceNameKey: dsname,
				common.FalconInstanceKey:     "kernel_sensor",
				common.FalconComponentKey:    "kernel_sensor",
				common.FalconManagedByKey:    dsname,
				common.FalconProviderKey:     "CrowdStrike",
				common.FalconPartOfKey:       "Falcon",
				common.FalconControllerKey:   "controller-manager",
			},
		},
		Data: containerData,
	}
}
