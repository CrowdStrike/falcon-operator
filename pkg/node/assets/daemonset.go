package assets

import (
	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
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

func pullSecrets(node *falconv1alpha1.FalconNodeSensor) []corev1.LocalObjectReference {
	if node.Spec.Node.ImageOverride == "" {
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

		/* Beta feature to enable later
		if node.Spec.Node.DSUpdateStrategy.RollingUpdate.MaxSurge != nil {
			rollingUpdateSettings.MaxSurge = node.Spec.Node.DSUpdateStrategy.RollingUpdate.MaxSurge
		}
		*/

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

func Daemonset(dsName string, image string, node *falconv1alpha1.FalconNodeSensor) *appsv1.DaemonSet {
	privileged := true
	escalation := true
	readOnlyFs := false
	hostpid := true
	hostnetwork := true
	hostipc := true
	runAs := int64(0)
	pathTypeUnset := corev1.HostPathUnset
	pathDirCreate := corev1.HostPathDirectoryOrCreate

	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dsName,
			Namespace: node.TargetNs(),
			Labels: map[string]string{
				common.FalconInstanceNameKey: dsName,
				common.FalconInstanceKey:     common.FalconKernelSensor,
				common.FalconComponentKey:    common.FalconKernelSensor,
				common.FalconManagedByKey:    node.Name,
				common.FalconProviderKey:     common.FalconProviderValue,
				common.FalconPartOfKey:       "Falcon",
				common.FalconControllerKey:   "controller-manager",
			},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					common.FalconInstanceNameKey: dsName,
					common.FalconInstanceKey:     common.FalconKernelSensor,
					common.FalconComponentKey:    common.FalconKernelSensor,
					common.FalconManagedByKey:    node.Name,
					common.FalconProviderKey:     common.FalconProviderValue,
					common.FalconPartOfKey:       "Falcon",
					common.FalconControllerKey:   "controller-manager",
				},
			},
			UpdateStrategy: dsUpdateStrategy(node),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						common.FalconInstanceNameKey: dsName,
						common.FalconInstanceKey:     common.FalconKernelSensor,
						common.FalconComponentKey:    common.FalconKernelSensor,
						common.FalconManagedByKey:    node.Name,
						common.FalconProviderKey:     common.FalconProviderValue,
						common.FalconPartOfKey:       "Falcon",
						common.FalconControllerKey:   "controller-manager",
					},
					Annotations: map[string]string{
						common.FalconContainerInjection: "disabled",
					},
				},
				Spec: corev1.PodSpec{
					// NodeSelector is set to linux until windows containers are supported for the Falcon sensor
					NodeSelector:                  common.NodeSelector,
					Tolerations:                   node.Spec.Node.Tolerations,
					HostPID:                       hostpid,
					HostIPC:                       hostipc,
					HostNetwork:                   hostnetwork,
					TerminationGracePeriodSeconds: getTermGracePeriod(node),
					ImagePullSecrets:              pullSecrets(node),
					InitContainers: []corev1.Container{
						{
							Name:    "init-falconstore",
							Image:   image,
							Command: common.FalconShellCommand,
							Args:    common.InitContainerArgs(),
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "falconstore-hostdir",
									MountPath: common.FalconHostInstallDir,
								},
							},
						},
					},
					ServiceAccountName: common.NodeServiceAccountName,
					Containers: []corev1.Container{
						{
							SecurityContext: &corev1.SecurityContext{
								Privileged:               &privileged,
								RunAsUser:                &runAs,
								ReadOnlyRootFilesystem:   &readOnlyFs,
								AllowPrivilegeEscalation: &escalation,
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
						},
					},
					Volumes: []corev1.Volume{
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
					},
				},
			},
		},
	}
}

func RemoveNodeDirDaemonset(dsName string, image string, node *falconv1alpha1.FalconNodeSensor) *appsv1.DaemonSet {
	privileged := true
	escalation := true
	readOnlyFs := false
	runAs := int64(0)

	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dsName,
			Namespace: node.TargetNs(),
			Labels: map[string]string{
				common.FalconInstanceNameKey: dsName,
				common.FalconInstanceKey:     "cleanup",
				common.FalconComponentKey:    "kernel_sensor",
				common.FalconManagedByKey:    node.Name,
				common.FalconProviderKey:     common.FalconProviderValue,
				common.FalconPartOfKey:       "Falcon",
				common.FalconControllerKey:   "controller-manager",
			},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					common.FalconInstanceNameKey: dsName,
					common.FalconInstanceKey:     "cleanup",
					common.FalconComponentKey:    "kernel_sensor",
					common.FalconManagedByKey:    node.Name,
					common.FalconProviderKey:     common.FalconProviderValue,
					common.FalconPartOfKey:       "Falcon",
					common.FalconControllerKey:   "controller-manager",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						common.FalconInstanceNameKey: dsName,
						common.FalconInstanceKey:     "cleanup",
						common.FalconComponentKey:    "kernel_sensor",
						common.FalconManagedByKey:    node.Name,
						common.FalconProviderKey:     common.FalconProviderValue,
						common.FalconPartOfKey:       "Falcon",
						common.FalconControllerKey:   "controller-manager",
					},
					Annotations: map[string]string{
						common.FalconContainerInjection: "disabled",
					},
				},
				Spec: corev1.PodSpec{
					// NodeSelector is set to linux until windows containers are supported for the Falcon sensor
					NodeSelector:                  common.NodeSelector,
					Tolerations:                   node.Spec.Node.Tolerations,
					TerminationGracePeriodSeconds: getTermGracePeriod(node),
					ImagePullSecrets:              pullSecrets(node),
					InitContainers: []corev1.Container{
						{
							Name:    "cleanup-opt-crowdstrike",
							Image:   image,
							Command: common.FalconShellCommand,
							Args:    common.InitCleanupArgs(),
							SecurityContext: &corev1.SecurityContext{
								Privileged:               &privileged,
								RunAsUser:                &runAs,
								ReadOnlyRootFilesystem:   &readOnlyFs,
								AllowPrivilegeEscalation: &escalation,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "opt-crowdstrike",
									MountPath: common.FalconHostInstallDir,
								},
							},
						},
					},
					ServiceAccountName: common.NodeServiceAccountName,
					Containers: []corev1.Container{
						{
							SecurityContext: &corev1.SecurityContext{
								Privileged:               &privileged,
								RunAsUser:                &runAs,
								ReadOnlyRootFilesystem:   &readOnlyFs,
								AllowPrivilegeEscalation: &escalation,
							},
							Name:    "cleanup-sleep",
							Image:   image,
							Command: common.FalconShellCommand,
							Args:    common.CleanupSleep(),
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "opt-crowdstrike",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: common.FalconHostInstallDir,
								},
							},
						},
					},
				},
			},
		},
	}
}
