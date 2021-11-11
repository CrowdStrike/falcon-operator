package node

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

func Daemonset(dsName string, node *falconv1alpha1.FalconNodeSensor) *appsv1.DaemonSet {
	return daemonset(dsName, node)
}

func daemonset(dsName string, node *falconv1alpha1.FalconNodeSensor) *appsv1.DaemonSet {
	privileged := true
	escalation := true
	readOnlyFs := false
	hostpid := true
	hostnetwork := false
	hostipc := true
	runAs := int64(0)
	pathTypeUnset := corev1.HostPathUnset

	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dsName,
			Namespace: node.Namespace,
			Labels: map[string]string{
				common.FalconInstanceNameKey: dsName,
				common.FalconInstanceKey:     "kernel_sensor",
				common.FalconComponentKey:    "kernel_sensor",
				common.FalconManagedByKey:    node.Name,
				common.FalconProviderKey:     "CrowdStrike",
				common.FalconPartOfKey:       "Falcon",
				common.FalconControllerKey:   "controller-manager",
			},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					common.FalconInstanceNameKey: dsName,
					common.FalconInstanceKey:     "kernel_sensor",
					common.FalconComponentKey:    "kernel_sensor",
					common.FalconManagedByKey:    node.Name,
					common.FalconProviderKey:     "CrowdStrike",
					common.FalconPartOfKey:       "Falcon",
					common.FalconControllerKey:   "controller-manager",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						common.FalconInstanceNameKey: dsName,
						common.FalconInstanceKey:     "kernel_sensor",
						common.FalconComponentKey:    "kernel_sensor",
						common.FalconManagedByKey:    node.Name,
						common.FalconProviderKey:     "CrowdStrike",
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
					InitContainers: []corev1.Container{
						{
							Name:    "init-falconstore",
							Image:   common.GetFalconImage(node),
							Command: common.FalconShellCommand,
							Args:    common.InitContainerArgs(),
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "falconstore-dir",
									MountPath: common.FalconDataDir,
								},
							},
						},
					},
					Containers: []corev1.Container{
						{
							SecurityContext: &corev1.SecurityContext{
								Privileged:               &privileged,
								RunAsUser:                &runAs,
								ReadOnlyRootFilesystem:   &readOnlyFs,
								AllowPrivilegeEscalation: &escalation,
							},
							Name:            "falcon-node-sensor",
							Image:           common.GetFalconImage(node),
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
									Name:      "dev",
									MountPath: "/dev",
								},
								{
									Name:      "etc",
									MountPath: "/etc",
								},
								{
									Name:      "var-log",
									MountPath: "/var/log",
								},
								{
									Name:      "var-run",
									MountPath: "/var/run",
								},
								{
									Name:      "falconstore",
									MountPath: "/opt/CrowdStrike/falconstore",
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "dev",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/dev",
									Type: &pathTypeUnset,
								},
							},
						},
						{
							Name: "etc",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/etc",
									Type: &pathTypeUnset,
								},
							},
						},
						{
							Name: "var-log",
							VolumeSource: corev1.VolumeSource{
								EmptyDir: &corev1.EmptyDirVolumeSource{},
							},
						},
						{
							Name: "var-run",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "/var/run",
									Type: &pathTypeUnset,
								},
							},
						},
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
							Name: "falconstore-dir",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: common.FalconDataDir,
									Type: &pathTypeUnset,
								},
							},
						},
					},
				},
			},
		},
	}
}
