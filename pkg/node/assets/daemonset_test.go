package assets

import (
	"testing"

	"github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestGetTermGracePeriod(t *testing.T) {
	falconNode := v1alpha1.FalconNodeSensor{}
	falconNode.Spec.FalconAPI = nil

	want := int64(10)
	got := getTermGracePeriod(&falconNode)
	if diff := cmp.Diff(&want, got); diff != "" {
		t.Errorf("getTermGracePeriod() mismatch (-want +got): %s", diff)
	}

	falconNode.Spec.Node.TerminationGracePeriod = want
	got = getTermGracePeriod(&falconNode)
	if diff := cmp.Diff(&want, got); diff != "" {
		t.Errorf("getTermGracePeriod() mismatch (-want +got): %s", diff)
	}
}

func TestPullSecrets(t *testing.T) {
	falconNode := v1alpha1.FalconNodeSensor{}

	want := []corev1.LocalObjectReference{
		{
			Name: common.FalconPullSecretName,
		},
	}

	got := pullSecrets(&falconNode)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("PullSecrets() mismatch (-want +got): %s", diff)
	}

	want = []corev1.LocalObjectReference{
		{
			Name: "testSecretName",
		},
	}

	falconNode.Spec.Node.ImageOverride = "testImageName"
	falconNode.Spec.Node.ImagePullSecrets = want

	got = pullSecrets(&falconNode)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("PullSecrets() mismatch (-want +got): %s", diff)
	}
}

func TestDsUpdateStrategy(t *testing.T) {
	falconNode := v1alpha1.FalconNodeSensor{}

	// Test OnDelete return value
	falconNode.Spec.Node.DSUpdateStrategy.Type = "OnDelete"
	got := dsUpdateStrategy(&falconNode)
	want := appsv1.DaemonSetUpdateStrategy{Type: appsv1.OnDeleteDaemonSetStrategyType}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("dsUpdateStrategy() mismatch (-want +got): %s", diff)
	}

	// Test RollingUpdate return value
	want = appsv1.DaemonSetUpdateStrategy{
		Type: appsv1.RollingUpdateDaemonSetStrategyType,
		RollingUpdate: &appsv1.RollingUpdateDaemonSet{
			MaxUnavailable: &intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: 1,
			},
		},
	}

	falconNode.Spec.Node.DSUpdateStrategy.Type = appsv1.RollingUpdateDaemonSetStrategyType
	falconNode.Spec.Node.DSUpdateStrategy.RollingUpdate.MaxUnavailable = &intstr.IntOrString{Type: intstr.Int, IntVal: 1}

	got = dsUpdateStrategy(&falconNode)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("dsUpdateStrategy() mismatch (-want +got): %s", diff)
	}
}

func TestDaemonset(t *testing.T) {
	falconNode := v1alpha1.FalconNodeSensor{}
	falconNode.Namespace = "falcon-system"
	falconNode.Name = "test"
	image := "testImage"
	dsName := "test-DaemonSet"

	privileged := true
	escalation := true
	readOnlyFs := false
	hostpid := true
	hostnetwork := true
	hostipc := true
	runAs := int64(0)
	pathTypeUnset := corev1.HostPathUnset
	pathDirCreate := corev1.HostPathDirectoryOrCreate

	want := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-DaemonSet",
			Namespace: falconNode.Namespace,
			Labels: map[string]string{
				common.FalconInstanceNameKey: "test-DaemonSet",
				common.FalconInstanceKey:     common.FalconKernelSensor,
				common.FalconComponentKey:    common.FalconKernelSensor,
				common.FalconManagedByKey:    "test",
				common.FalconProviderKey:     common.FalconProviderValue,
				common.FalconPartOfKey:       "Falcon",
				common.FalconControllerKey:   "controller-manager",
			},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					common.FalconInstanceNameKey: "test-DaemonSet",
					common.FalconInstanceKey:     common.FalconKernelSensor,
					common.FalconComponentKey:    common.FalconKernelSensor,
					common.FalconManagedByKey:    "test",
					common.FalconProviderKey:     common.FalconProviderValue,
					common.FalconPartOfKey:       "Falcon",
					common.FalconControllerKey:   "controller-manager",
				},
			},
			UpdateStrategy: dsUpdateStrategy(&falconNode),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						common.FalconInstanceNameKey: "test-DaemonSet",
						common.FalconInstanceKey:     common.FalconKernelSensor,
						common.FalconComponentKey:    common.FalconKernelSensor,
						common.FalconManagedByKey:    "test",
						common.FalconProviderKey:     common.FalconProviderValue,
						common.FalconPartOfKey:       "Falcon",
						common.FalconControllerKey:   "controller-manager",
					},
					Annotations: map[string]string{"sensor.falcon-system.crowdstrike.com/injection": "disabled"},
				},
				Spec: corev1.PodSpec{
					// NodeSelector is set to linux until windows containers are supported for the Falcon sensor
					NodeSelector:                  common.NodeSelector,
					Tolerations:                   falconNode.Spec.Node.Tolerations,
					HostPID:                       hostpid,
					HostIPC:                       hostipc,
					HostNetwork:                   hostnetwork,
					TerminationGracePeriodSeconds: getTermGracePeriod(&falconNode),
					ImagePullSecrets:              pullSecrets(&falconNode),
					InitContainers: []corev1.Container{
						{
							Name:    "init-falconstore",
							Image:   image,
							Command: common.FalconShellCommand,
							Args:    common.InitContainerArgs(),
							SecurityContext: &corev1.SecurityContext{
								Privileged:               &privileged,
								RunAsUser:                &runAs,
								ReadOnlyRootFilesystem:   &readOnlyFs,
								AllowPrivilegeEscalation: &escalation,
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "falconstore-hostdir",
									MountPath: common.FalconInitHostInstallDir,
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
							ImagePullPolicy: falconNode.Spec.Node.ImagePullPolicy,
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

	got := Daemonset(dsName, image, common.NodeServiceAccountName, &falconNode)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Daemonset() mismatch (-want +got): %s", diff)
	}
}

func TestRemoveNodeDirDaemonset(t *testing.T) {
	falconNode := v1alpha1.FalconNodeSensor{}
	falconNode.Namespace = "falcon-system"
	falconNode.Name = "test"
	image := "testImage"
	dsName := "test-DaemonSet"

	privileged := true
	escalation := true
	readOnlyFs := false
	hostpid := true
	hostnetwork := true
	hostipc := true
	runAs := int64(0)

	want := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dsName,
			Namespace: falconNode.Namespace,
			Labels: map[string]string{
				common.FalconInstanceNameKey: "test-DaemonSet",
				common.FalconInstanceKey:     "cleanup",
				common.FalconComponentKey:    common.FalconKernelSensor,
				common.FalconManagedByKey:    "test",
				common.FalconProviderKey:     common.FalconProviderValue,
				common.FalconPartOfKey:       "Falcon",
				common.FalconControllerKey:   "controller-manager",
			},
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					common.FalconInstanceNameKey: "test-DaemonSet",
					common.FalconInstanceKey:     "cleanup",
					common.FalconComponentKey:    common.FalconKernelSensor,
					common.FalconManagedByKey:    "test",
					common.FalconProviderKey:     common.FalconProviderValue,
					common.FalconPartOfKey:       "Falcon",
					common.FalconControllerKey:   "controller-manager",
				},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						common.FalconInstanceNameKey: "test-DaemonSet",
						common.FalconInstanceKey:     "cleanup",
						common.FalconComponentKey:    common.FalconKernelSensor,
						common.FalconManagedByKey:    "test",
						common.FalconProviderKey:     common.FalconProviderValue,
						common.FalconPartOfKey:       "Falcon",
						common.FalconControllerKey:   "controller-manager",
					},
					Annotations: map[string]string{"sensor.falcon-system.crowdstrike.com/injection": "disabled"},
				},
				Spec: corev1.PodSpec{
					// NodeSelector is set to linux until windows containers are supported for the Falcon sensor
					NodeSelector:                  common.NodeSelector,
					Tolerations:                   falconNode.Spec.Node.Tolerations,
					HostPID:                       hostpid,
					HostIPC:                       hostipc,
					HostNetwork:                   hostnetwork,
					TerminationGracePeriodSeconds: getTermGracePeriod(&falconNode),
					ImagePullSecrets:              pullSecrets(&falconNode),
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
									MountPath: common.FalconInitHostInstallDir,
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

	got := RemoveNodeDirDaemonset(dsName, image, common.NodeServiceAccountName, &falconNode)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Daemonset() mismatch (-want +got): %s", diff)
	}
}
