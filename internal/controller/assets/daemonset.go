package assets

import (
	"reflect"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func getTermGracePeriod(node *falconv1alpha1.FalconNodeSensor) *int64 {
	gracePeriod := node.Spec.Node.TerminationGracePeriod
	if gracePeriod < 10 {
		gracePeriod = 10
	}
	gp := int64(gracePeriod)
	return &gp

}

func nodeAffinity(node *falconv1alpha1.FalconNodeSensor) *corev1.Affinity {
	if !reflect.DeepEqual(node.Spec.Node.NodeAffinity, corev1.NodeAffinity{}) {
		return &corev1.Affinity{NodeAffinity: &node.Spec.Node.NodeAffinity}
	}
	return &corev1.Affinity{}
}

func pullSecrets(node *falconv1alpha1.FalconNodeSensor) []corev1.LocalObjectReference {
	if node.Spec.Node.Image == "" {
		return []corev1.LocalObjectReference{
			{
				Name: common.FalconPullSecretName,
			},
		}
	} else {
		return node.Spec.Node.ImagePullSecrets
	}
}

func dsUpdateStrategy(node *falconv1alpha1.FalconNodeSensor) appsv1.DaemonSetUpdateStrategy {
	if node.Spec.Node.DSUpdateStrategy.Type == appsv1.RollingUpdateDaemonSetStrategyType || node.Spec.Node.DSUpdateStrategy.Type == "" {
		rollingUpdateSettings := appsv1.RollingUpdateDaemonSet{}

		if node.Spec.Node.DSUpdateStrategy.RollingUpdate.MaxSurge != nil {
			rollingUpdateSettings.MaxSurge = node.Spec.Node.DSUpdateStrategy.RollingUpdate.MaxSurge
		}

		if node.Spec.Node.DSUpdateStrategy.RollingUpdate.MaxUnavailable != nil {
			rollingUpdateSettings.MaxUnavailable = node.Spec.Node.DSUpdateStrategy.RollingUpdate.MaxUnavailable
		}

		return appsv1.DaemonSetUpdateStrategy{
			Type:          appsv1.RollingUpdateDaemonSetStrategyType,
			RollingUpdate: &rollingUpdateSettings,
		}
	}

	return appsv1.DaemonSetUpdateStrategy{Type: appsv1.OnDeleteDaemonSetStrategyType}
}

func sensorCapabilities(node *falconv1alpha1.FalconNodeSensor, initContainer bool) *corev1.Capabilities {
	if node.Spec.Node.GKE.Enabled != nil && *node.Spec.Node.GKE.Enabled {
		if initContainer {
			return &corev1.Capabilities{
				Add: []corev1.Capability{
					"SYS_ADMIN",
					"SYS_PTRACE",
					"SYS_CHROOT",
					"DAC_READ_SEARCH",
				},
			}
		}
		return &corev1.Capabilities{
			Add: []corev1.Capability{
				"SYS_ADMIN",
				"SETGID",
				"SETUID",
				"SYS_PTRACE",
				"SYS_CHROOT",
				"DAC_OVERRIDE",
				"SETPCAP",
				"DAC_READ_SEARCH",
				"BPF",
				"PERFMON",
				"SYS_RESOURCE",
				"NET_RAW",
				"CHOWN",
			},
		}
	}
	return nil
}

func initContainerResources(node *falconv1alpha1.FalconNodeSensor) corev1.ResourceRequirements {
	if node.Spec.Node.Backend == "bpf" && (node.Spec.Node.SensorResources != falconv1alpha1.Resources{} || (node.Spec.Node.GKE.Enabled != nil && *node.Spec.Node.GKE.Enabled)) {
		return corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				"cpu":               resource.MustParse("10m"),
				"ephemeral-storage": resource.MustParse("100Mi"),
				"memory":            resource.MustParse("50Mi"),
			},
			Requests: corev1.ResourceList{
				"cpu":               resource.MustParse("10m"),
				"ephemeral-storage": resource.MustParse("100Mi"),
				"memory":            resource.MustParse("50Mi"),
			},
			Claims: []corev1.ResourceClaim{},
		}
	}

	return corev1.ResourceRequirements{}
}

func dsResources(node *falconv1alpha1.FalconNodeSensor) corev1.ResourceRequirements {
	if node.Spec.Node.Backend == "bpf" {
		limitResources := corev1.ResourceList{}
		requestsResources := corev1.ResourceList{}

		if node.Spec.Node.GKE.Enabled != nil && *node.Spec.Node.GKE.Enabled {
			limitResources = corev1.ResourceList{
				"cpu":               resource.MustParse("750m"),
				"memory":            resource.MustParse("1.5Gi"),
				"ephemeral-storage": resource.MustParse("100Mi"),
			}
			requestsResources = corev1.ResourceList{
				"cpu":               resource.MustParse("750m"),
				"memory":            resource.MustParse("1.5Gi"),
				"ephemeral-storage": resource.MustParse("100Mi"),
			}
		}

		if node.Spec.Node.SensorResources.Limits.CPU != "" {
			limitResources["cpu"] = resource.MustParse(node.Spec.Node.SensorResources.Limits.CPU)
		}
		if node.Spec.Node.SensorResources.Limits.Memory != "" {
			limitResources["memory"] = resource.MustParse(node.Spec.Node.SensorResources.Limits.Memory)
		}
		if node.Spec.Node.SensorResources.Limits.EphemeralStorage != "" {
			limitResources["ephemeral-storage"] = resource.MustParse(node.Spec.Node.SensorResources.Limits.EphemeralStorage)
		}
		if node.Spec.Node.SensorResources.Requests.CPU != "" {
			requestsResources["cpu"] = resource.MustParse(node.Spec.Node.SensorResources.Requests.CPU)
		}
		if node.Spec.Node.SensorResources.Requests.EphemeralStorage != "" {
			requestsResources["ephemeral-storage"] = resource.MustParse(node.Spec.Node.SensorResources.Requests.EphemeralStorage)
		}
		if node.Spec.Node.SensorResources.Requests.Memory != "" {
			requestsResources["memory"] = resource.MustParse(node.Spec.Node.SensorResources.Requests.Memory)
		}

		return corev1.ResourceRequirements{
			Limits:   limitResources,
			Requests: requestsResources,
			Claims:   []corev1.ResourceClaim{},
		}
	}

	node.Spec.Node.SensorResources = falconv1alpha1.Resources{}
	return corev1.ResourceRequirements{}
}

// initArgs - remove this function when 6.53 is no longer supported. 6.53+ will use InitContainerArgs()
func initArgs(node *falconv1alpha1.FalconNodeSensor) []string {
	if node.Spec.Node.GKE.Enabled != nil && *node.Spec.Node.GKE.Enabled {
		return common.InitContainerArgs()
	}
	return common.LegacyInitContainerArgs()
}

// cleanupArgs - remove this function when 6.53 is no longer supported. 6.53+ will use InitCleanupArgs()
func cleanupArgs(node *falconv1alpha1.FalconNodeSensor) []string {
	if node.Spec.Node.GKE.Enabled != nil && *node.Spec.Node.GKE.Enabled {
		return common.InitCleanupArgs()
	}
	return common.LegacyInitCleanupArgs()
}

// volumes - remove this function when 6.53 is no longer supported. 6.53+ will only use falconstore
func volumes(node *falconv1alpha1.FalconNodeSensor) []corev1.Volume {
	pathTypeUnset := corev1.HostPathUnset
	pathDirCreate := corev1.HostPathDirectoryOrCreate

	if node.Spec.Node.GKE.Enabled != nil && *node.Spec.Node.GKE.Enabled {
		return []corev1.Volume{
			{
				Name: "falconstore",
				VolumeSource: corev1.VolumeSource{
					HostPath: &corev1.HostPathVolumeSource{
						Path: common.FalconStoreFile,
						Type: &pathTypeUnset,
					},
				},
			},
		}
	}

	return []corev1.Volume{
		{
			Name: "falconstore",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: common.FalconStoreFile,
					Type: &pathTypeUnset,
				},
			},
		},
		{
			Name: "falconstore-hostdir",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: common.FalconHostInstallDir,
					Type: &pathDirCreate,
				},
			},
		},
	}
}

func volumeMounts(node *falconv1alpha1.FalconNodeSensor, name string) []corev1.VolumeMount {
	if node.Spec.Node.GKE.Enabled != nil && *node.Spec.Node.GKE.Enabled {
		return []corev1.VolumeMount{}
	}

	return []corev1.VolumeMount{
		{
			Name:      name,
			MountPath: common.FalconInitHostInstallDir,
		},
	}
}

func volumesCleanup(node *falconv1alpha1.FalconNodeSensor) []corev1.Volume {
	if node.Spec.Node.GKE.Enabled != nil && *node.Spec.Node.GKE.Enabled {
		return []corev1.Volume{}
	}
	return []corev1.Volume{
		{
			Name: "opt-crowdstrike",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: common.FalconHostInstallDir,
				},
			},
		},
	}

}

func Daemonset(dsName, image, serviceAccount string, node *falconv1alpha1.FalconNodeSensor) *appsv1.DaemonSet {
	privileged := true
	escalation := true
	readOnlyFSDisabled := false
	readOnlyFSEnabled := true
	hostpid := true
	hostnetwork := true
	hostipc := true
	runAsRoot := int64(0)

	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dsName,
			Namespace: node.TargetNs(),
			Labels:    common.CRLabels("daemonset", dsName, common.FalconKernelSensor),
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: common.CRLabels("daemonset", dsName, common.FalconKernelSensor),
			},
			UpdateStrategy: dsUpdateStrategy(node),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: common.CRLabels("daemonset", dsName, common.FalconKernelSensor),
					Annotations: map[string]string{
						common.FalconContainerInjection: "disabled",
					},
				},
				Spec: corev1.PodSpec{
					// NodeSelector is set to linux until windows containers are supported for the Falcon sensor
					NodeSelector:                  common.NodeSelector,
					Affinity:                      nodeAffinity(node),
					Tolerations:                   node.Spec.Node.Tolerations,
					HostPID:                       hostpid,
					HostIPC:                       hostipc,
					HostNetwork:                   hostnetwork,
					TerminationGracePeriodSeconds: getTermGracePeriod(node),
					ImagePullSecrets:              pullSecrets(node),
					InitContainers: []corev1.Container{
						{
							Name:         "init-falconstore",
							Image:        image,
							Command:      common.FalconShellCommand,
							Args:         initArgs(node),
							VolumeMounts: volumeMounts(node, "falconstore-hostdir"),
							Resources:    initContainerResources(node),
							SecurityContext: &corev1.SecurityContext{
								Privileged:               &privileged,
								RunAsUser:                &runAsRoot,
								ReadOnlyRootFilesystem:   &readOnlyFSEnabled,
								AllowPrivilegeEscalation: &escalation,
								Capabilities:             sensorCapabilities(node, true),
							},
						},
					},
					ServiceAccountName: serviceAccount,
					Containers: []corev1.Container{
						{
							SecurityContext: &corev1.SecurityContext{
								Privileged:               &privileged,
								RunAsUser:                &runAsRoot,
								ReadOnlyRootFilesystem:   &readOnlyFSDisabled,
								AllowPrivilegeEscalation: &escalation,
								Capabilities:             sensorCapabilities(node, false),
							},
							Name:            "falcon-node-sensor",
							Image:           image,
							ImagePullPolicy: node.Spec.Node.ImagePullPolicy,
							EnvFrom: []corev1.EnvFromSource{
								{
									ConfigMapRef: &corev1.ConfigMapEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: dsName + "-config",
										},
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "falconstore",
									MountPath: common.FalconStoreFile,
								},
							},
							Resources: dsResources(node),
						},
					},
					Volumes:           volumes(node),
					PriorityClassName: node.Spec.Node.PriorityClass.Name,
				},
			},
		},
	}
}

func RemoveNodeDirDaemonset(dsName, image, serviceAccount string, node *falconv1alpha1.FalconNodeSensor) *appsv1.DaemonSet {
	privileged := true
	nonPrivileged := false
	escalation := true
	allowEscalation := false
	readOnlyFs := true
	hostpid := true
	runAsRoot := int64(0)

	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dsName,
			Namespace: node.TargetNs(),
			Labels:    common.CRLabels("cleanup", dsName, common.FalconKernelSensor),
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: common.CRLabels("cleanup", dsName, common.FalconKernelSensor),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: common.CRLabels("cleanup", dsName, common.FalconKernelSensor),
					Annotations: map[string]string{
						common.FalconContainerInjection: "disabled",
					},
				},
				Spec: corev1.PodSpec{
					// NodeSelector is set to linux until windows containers are supported for the Falcon sensor
					NodeSelector:                  common.NodeSelector,
					Affinity:                      nodeAffinity(node),
					Tolerations:                   node.Spec.Node.Tolerations,
					HostPID:                       hostpid,
					TerminationGracePeriodSeconds: getTermGracePeriod(node),
					ImagePullSecrets:              pullSecrets(node),
					InitContainers: []corev1.Container{
						{
							Name:      "cleanup-opt-crowdstrike",
							Image:     image,
							Command:   common.FalconShellCommand,
							Args:      cleanupArgs(node),
							Resources: initContainerResources(node),
							SecurityContext: &corev1.SecurityContext{
								Privileged:               &privileged,
								RunAsUser:                &runAsRoot,
								ReadOnlyRootFilesystem:   &readOnlyFs,
								AllowPrivilegeEscalation: &escalation,
								Capabilities:             sensorCapabilities(node, true),
							},
							VolumeMounts: volumeMounts(node, "opt-crowdstrike"),
						},
					},
					ServiceAccountName: serviceAccount,
					Containers: []corev1.Container{
						{
							Name:      "cleanup-sleep",
							Image:     image,
							Command:   common.FalconShellCommand,
							Args:      common.CleanupSleep(),
							Resources: initContainerResources(node),
							SecurityContext: &corev1.SecurityContext{
								Privileged:               &nonPrivileged,
								ReadOnlyRootFilesystem:   &readOnlyFs,
								AllowPrivilegeEscalation: &allowEscalation,
							},
						},
					},
					Volumes: volumesCleanup(node),
				},
			},
		},
	}
}
