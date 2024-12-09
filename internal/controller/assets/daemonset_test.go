package assets

import (
	"testing"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func TestGetTermGracePeriod(t *testing.T) {
	falconNode := falconv1alpha1.FalconNodeSensor{}
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

func TestNodeAffinity(t *testing.T) {
	falconNode := falconv1alpha1.FalconNodeSensor{}

	got := nodeAffinity(&falconNode)
	want := &corev1.Affinity{}
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("nodeAffinity() mismatch (-want +got): %s", diff)
	}

	testAffinity := &corev1.NodeAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
			NodeSelectorTerms: []corev1.NodeSelectorTerm{
				{
					MatchExpressions: []corev1.NodeSelectorRequirement{
						{
							Key:      "kubernetes.io/arch",
							Operator: corev1.NodeSelectorOpIn,
							Values:   []string{"amd64", "arm64"},
						},
					},
				},
			},
		},
	}

	falconNode.Spec.Node.NodeAffinity = *testAffinity
	want = &corev1.Affinity{NodeAffinity: testAffinity}

	got = nodeAffinity(&falconNode)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("nodeAffinity() mismatch (-want +got): %s", diff)
	}
}

func TestPullSecrets(t *testing.T) {
	falconNode := falconv1alpha1.FalconNodeSensor{}

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

	falconNode.Spec.Node.Image = "testImageName"
	falconNode.Spec.Node.ImagePullSecrets = want

	got = pullSecrets(&falconNode)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("PullSecrets() mismatch (-want +got): %s", diff)
	}
}

func TestDsUpdateStrategy(t *testing.T) {
	falconNode := falconv1alpha1.FalconNodeSensor{}

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
			MaxSurge: &intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: 1,
			},
			MaxUnavailable: &intstr.IntOrString{
				Type:   intstr.Int,
				IntVal: 1,
			},
		},
	}

	falconNode.Spec.Node.DSUpdateStrategy.Type = appsv1.RollingUpdateDaemonSetStrategyType
	falconNode.Spec.Node.DSUpdateStrategy.RollingUpdate.MaxUnavailable = &intstr.IntOrString{Type: intstr.Int, IntVal: 1}
	falconNode.Spec.Node.DSUpdateStrategy.RollingUpdate.MaxSurge = &intstr.IntOrString{Type: intstr.Int, IntVal: 1}

	got = dsUpdateStrategy(&falconNode)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("dsUpdateStrategy() mismatch (-want +got): %s", diff)
	}
}

func TestSensorCapabilities(t *testing.T) {
	falconNode := falconv1alpha1.FalconNodeSensor{}
	enabled := true

	// Check default return value when GKE is not enabled
	got := sensorCapabilities(&falconNode, false)
	wantNil := (*corev1.Capabilities)(nil)
	if diff := cmp.Diff(wantNil, got); diff != "" {
		t.Errorf("sensorCapabilities() mismatch (-want +got): %s", diff)
	}

	// Check return value when GKE is enabled
	falconNode.Spec.Node.GKE.Enabled = &enabled
	got = sensorCapabilities(&falconNode, false)
	want := &corev1.Capabilities{
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
			"NET_ADMIN",
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("sensorCapabilities() mismatch (-want +got): %s", diff)
	}

	// check if initContainer is enabled
	got = sensorCapabilities(&falconNode, true)
	want = &corev1.Capabilities{
		Add: []corev1.Capability{
			"SYS_ADMIN",
			"SYS_PTRACE",
			"SYS_CHROOT",
			"DAC_READ_SEARCH",
		},
	}

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("sensorCapabilities() mismatch (-want +got): %s", diff)
	}
}

func TestDaemonset(t *testing.T) {
	falconNode := falconv1alpha1.FalconNodeSensor{}
	falconNode.Namespace = "falcon-system"
	falconNode.Name = "test"
	image := "testImage"
	dsName := "test-DaemonSet"
	falconNode.Spec.Node.Tolerations = &[]corev1.Toleration{
		{
			Key:      "node-role.kubernetes.io/master",
			Operator: "Exists",
			Effect:   "NoSchedule",
		},
		{
			Key:      "node-role.kubernetes.io/control-plane",
			Operator: "Exists",
			Effect:   "NoSchedule",
		},
		{
			Key:      "node-role.kubernetes.io/infra",
			Operator: "Exists",
			Effect:   "NoSchedule",
		},
	}

	privileged := true
	escalation := true
	readOnlyFSDisabled := false
	readOnlyFSEnabled := true
	hostpid := true
	hostnetwork := true
	hostipc := true
	runAsRoot := int64(0)
	pathTypeUnset := corev1.HostPathUnset

	want := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dsName,
			Namespace: falconNode.Spec.InstallNamespace,
			Labels:    common.CRLabels("daemonset", dsName, common.FalconKernelSensor),
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: common.CRLabels("daemonset", dsName, common.FalconKernelSensor),
			},
			UpdateStrategy: dsUpdateStrategy(&falconNode),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      common.CRLabels("daemonset", dsName, common.FalconKernelSensor),
					Annotations: map[string]string{"sensor.falcon-system.crowdstrike.com/injection": "disabled"},
				},
				Spec: corev1.PodSpec{
					// NodeSelector is set to linux until windows containers are supported for the Falcon sensor
					NodeSelector:                  common.NodeSelector,
					Affinity:                      nodeAffinity(&falconNode),
					Tolerations:                   *falconNode.Spec.Node.Tolerations,
					HostPID:                       hostpid,
					HostIPC:                       hostipc,
					HostNetwork:                   hostnetwork,
					TerminationGracePeriodSeconds: getTermGracePeriod(&falconNode),
					ImagePullSecrets:              pullSecrets(&falconNode),
					InitContainers: []corev1.Container{
						{
							Name:      "init-falconstore",
							Image:     image,
							Command:   common.FalconShellCommand,
							Args:      common.InitContainerArgs(),
							Resources: initContainerResources(&falconNode),
							SecurityContext: &corev1.SecurityContext{
								Privileged:               &privileged,
								RunAsUser:                &runAsRoot,
								ReadOnlyRootFilesystem:   &readOnlyFSEnabled,
								AllowPrivilegeEscalation: &escalation,
							},
						},
					},
					ServiceAccountName: common.NodeServiceAccountName,
					Containers: []corev1.Container{
						{
							SecurityContext: &corev1.SecurityContext{
								Privileged:               &privileged,
								RunAsUser:                &runAsRoot,
								ReadOnlyRootFilesystem:   &readOnlyFSDisabled,
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
							Resources: dsResources(&falconNode),
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
	falconNode := falconv1alpha1.FalconNodeSensor{}
	falconNode.Namespace = "falcon-system"
	falconNode.Name = "test"
	image := "testImage"
	dsName := "test-DaemonSet"
	falconNode.Spec.Node.Tolerations = &[]corev1.Toleration{
		{
			Key:      "node-role.kubernetes.io/master",
			Operator: "Exists",
			Effect:   "NoSchedule",
		},
		{
			Key:      "node-role.kubernetes.io/control-plane",
			Operator: "Exists",
			Effect:   "NoSchedule",
		},
		{
			Key:      "node-role.kubernetes.io/infra",
			Operator: "Exists",
			Effect:   "NoSchedule",
		},
	}

	privileged := true
	nonPrivileged := false
	escalation := true
	allowEscalation := false
	readOnlyFs := true
	hostpid := true
	runAsRoot := int64(0)

	want := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dsName,
			Namespace: falconNode.Spec.InstallNamespace,
			Labels:    common.CRLabels("cleanup", dsName, common.FalconKernelSensor),
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: common.CRLabels("cleanup", dsName, common.FalconKernelSensor),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      common.CRLabels("cleanup", dsName, common.FalconKernelSensor),
					Annotations: map[string]string{common.FalconContainerInjection: "disabled"},
				},
				Spec: corev1.PodSpec{
					// NodeSelector is set to linux until windows containers are supported for the Falcon sensor
					NodeSelector:                  common.NodeSelector,
					Affinity:                      nodeAffinity(&falconNode),
					Tolerations:                   *falconNode.Spec.Node.Tolerations,
					HostPID:                       hostpid,
					TerminationGracePeriodSeconds: getTermGracePeriod(&falconNode),
					ImagePullSecrets:              pullSecrets(&falconNode),
					InitContainers: []corev1.Container{
						{
							Name:      "cleanup-opt-crowdstrike",
							Image:     image,
							Command:   common.FalconShellCommand,
							Args:      common.InitCleanupArgs(),
							Resources: initContainerResources(&falconNode),
							SecurityContext: &corev1.SecurityContext{
								Privileged:               &privileged,
								RunAsUser:                &runAsRoot,
								ReadOnlyRootFilesystem:   &readOnlyFs,
								AllowPrivilegeEscalation: &escalation,
							},
						},
					},
					ServiceAccountName: common.NodeServiceAccountName,
					Containers: []corev1.Container{
						{
							Name:      "cleanup-sleep",
							Image:     image,
							Command:   common.FalconShellCommand,
							Args:      common.CleanupSleep(),
							Resources: initContainerResources(&falconNode),
							SecurityContext: &corev1.SecurityContext{
								Privileged:               &nonPrivileged,
								ReadOnlyRootFilesystem:   &readOnlyFs,
								AllowPrivilegeEscalation: &allowEscalation,
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
