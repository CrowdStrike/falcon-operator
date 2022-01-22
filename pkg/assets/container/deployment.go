package container

import (
	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func ContainerDeployment(dsName string, nsName string, falconContainer *falconv1alpha1.FalconContainer) *appsv1.Deployment {
	replicas := falconContainer.Spec.FalconContainerSensorConfig.Replicas
	runNonRoot := true
	injectorPort := falconContainer.Spec.FalconContainerSensorConfig.InjectorPort
	image := falconContainer.Spec.FalconContainerSensorConfig.Image
	pullPolicy := falconContainer.Spec.FalconContainerSensorConfig.ImagePullPolicy
	recpu := falconContainer.Spec.FalconContainerSensorConfig.InjectorResources.Requests.CPU
	remem := falconContainer.Spec.FalconContainerSensorConfig.InjectorResources.Requests.Memory
	licpu := falconContainer.Spec.FalconContainerSensorConfig.InjectorResources.Limits.CPU
	limem := falconContainer.Spec.FalconContainerSensorConfig.InjectorResources.Limits.CPU
	azurePath := falconContainer.Spec.FalconContainerSensorConfig.AzureConfig
	azure := falconContainer.Spec.FalconContainerSensorConfig.AzureEnable
	enablePullSecrets := falconContainer.Spec.FalconContainerSensorConfig.EnablePullSecrets

	labels := map[string]string{
		common.FalconInstanceNameKey: dsName,
		common.FalconInstanceKey:     "container_sensor",
		common.FalconComponentKey:    "container_sensor",
		common.FalconManagedByKey:    dsName,
		common.FalconProviderKey:     "CrowdStrike",
		common.FalconPartOfKey:       "Falcon",
		common.FalconControllerKey:   "controller-manager",
	}

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dsName,
			Namespace: nsName,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
				},
				Spec: corev1.PodSpec{
					Affinity: &corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
								NodeSelectorTerms: []corev1.NodeSelectorTerm{
									{
										MatchExpressions: []corev1.NodeSelectorRequirement{
											{
												Key:      "kubernetes.io/os",
												Operator: corev1.NodeSelectorOpIn,
												Values:   []string{"linux"},
											},
											{
												Key:      "node-role.kubernetes.io/master",
												Operator: corev1.NodeSelectorOpDoesNotExist,
											},
										},
									},
								},
							},
						},
					},
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: &runNonRoot,
					},
					ImagePullSecrets: imagePullSecret(dsName, enablePullSecrets),
					InitContainers:   initContainer(dsName, image, azure, pullPolicy),
					Containers: []corev1.Container{
						{
							Name:            dsName + "-falcon-sensor",
							Image:           image,
							ImagePullPolicy: pullPolicy,
							Command:         common.FalconInjectorCommand,
							EnvFrom: []corev1.EnvFromSource{
								{
									ConfigMapRef: &corev1.ConfigMapEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: dsName + "-config",
										},
									},
								},
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: injectorPort,
									Name:          common.FalconServiceHTTPSName,
								},
							},
							VolumeMounts: volumeMounts(dsName, azure),
							ReadinessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   common.FalconContainerProbePath,
										Port:   intstr.IntOrString{IntVal: injectorPort},
										Scheme: corev1.URISchemeHTTPS,
									},
								},
								InitialDelaySeconds: 5,
								TimeoutSeconds:      1,
								PeriodSeconds:       10,
								SuccessThreshold:    1,
								FailureThreshold:    3,
							},
							LivenessProbe: &corev1.Probe{
								Handler: corev1.Handler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   common.FalconContainerProbePath,
										Port:   intstr.IntOrString{IntVal: injectorPort},
										Scheme: corev1.URISchemeHTTPS,
									},
								},
								InitialDelaySeconds: 5,
								TimeoutSeconds:      1,
								PeriodSeconds:       10,
								SuccessThreshold:    1,
								FailureThreshold:    3,
							},
							Resources: corev1.ResourceRequirements{
								Requests: resources(recpu, remem),
								Limits:   resources(licpu, limem),
							},
						},
					},
					Volumes: volumes(dsName, azurePath, azure),
				},
			},
		},
	}
}

func resources(cpu string, mem string) corev1.ResourceList {
	res := corev1.ResourceList{}

	if cpu != "" {
		res[corev1.ResourceCPU] = resource.MustParse(cpu)
	}
	if mem != "" {
		res[corev1.ResourceMemory] = resource.MustParse(mem)
	}

	return res
}

func imagePullSecret(dsName string, enable bool) []corev1.LocalObjectReference {
	if !enable {
		return nil
	}

	return []corev1.LocalObjectReference{
		{
			Name: dsName + "-pull-secret",
		},
	}
}

func initContainer(dsName string, image string, azure bool, pullPolicy corev1.PullPolicy) []corev1.Container {
	runNonRoot := false
	runAsUser := int64(0)
	privileged := false

	if !azure {
		return nil
	}

	return []corev1.Container{
		{
			Name:            dsName + "-init-container",
			Image:           image,
			ImagePullPolicy: pullPolicy,
			Command:         []string{"bash", "-c", "cp /run/azure.json /tmp/CrowdStrike/; chmod a+r /tmp/CrowdStrike/azure.json"},
			SecurityContext: &corev1.SecurityContext{
				RunAsNonRoot: &runNonRoot,
				RunAsUser:    &runAsUser,
				Privileged:   &privileged,
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      dsName + "-volume",
					MountPath: "/tmp/CrowdStrike",
					ReadOnly:  true,
				},
				{
					Name:      dsName + "-azure-config",
					MountPath: "/run/azure.json",
					ReadOnly:  true,
				},
			},
		},
	}
}

func volumeMounts(dsName string, azure bool) []corev1.VolumeMount {
	if !azure {
		return []corev1.VolumeMount{
			{
				Name:      dsName + "-tls-certs",
				MountPath: "/run/secrets/tls",
				ReadOnly:  true,
			},
		}
	}

	return []corev1.VolumeMount{
		{
			Name:      dsName + "-tls-certs",
			MountPath: "/run/secrets/tls",
			ReadOnly:  true,
		},
		{
			Name:      dsName + "-volume",
			MountPath: "/tmp/CrowdStrike",
			ReadOnly:  true,
		},
	}

}

func volumes(dsName string, azurePath string, azure bool) []corev1.Volume {
	pathTypeFile := corev1.HostPathFile

	if !azure {
		return []corev1.Volume{
			{
				Name: dsName + "-tls-certs",
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName: dsName + "-tls",
					},
				},
			},
		}
	}

	return []corev1.Volume{
		{
			Name: dsName + "-tls-certs",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: dsName + "-tls",
				},
			},
		},
		{
			Name: dsName + "-volume",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: dsName + "-azure-config",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: azurePath,
					Type: &pathTypeFile,
				},
			},
		},
	}
}
