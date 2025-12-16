package assets

import (
	"fmt"
	"maps"
	"reflect"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	nobodyGroup = 65534
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
	if len(node.Spec.Node.ImagePullSecrets) == 0 {
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

func dsAutoPilotDeployAllowlistLabel(node *falconv1alpha1.FalconNodeSensor) map[string]string {
	if *node.Spec.Node.GKE.Enabled && node.Spec.Node.GKE.DeployAllowListVersion != nil {
		return map[string]string{
			common.GKEAutoPilotAllowListLabelKey: fmt.Sprintf("%s-%s", common.GKEAutoPilotDeployDSAllowlistPrefix, *node.Spec.Node.GKE.DeployAllowListVersion),
		}
	}
	return nil
}

func dsAutoPilotCleanupAllowlistLabel(node *falconv1alpha1.FalconNodeSensor) map[string]string {
	if *node.Spec.Node.GKE.Enabled && node.Spec.Node.GKE.CleanupAllowListVersion != nil {
		return map[string]string{
			common.GKEAutoPilotAllowListLabelKey: fmt.Sprintf("%s-%s", common.GKEAutoPilotCleanupAllowlistPrefix, *node.Spec.Node.GKE.CleanupAllowListVersion),
		}
	}
	return nil
}

func dsManageAutoPilotLabels(dsType string, dsName string, f func(*falconv1alpha1.FalconNodeSensor) map[string]string, node *falconv1alpha1.FalconNodeSensor) map[string]string {
	dsLabels := common.CRLabels(dsType, dsName, common.FalconKernelSensor)
	autoPilotLabel := f(node)

	if autoPilotLabel != nil {
		maps.Copy(dsLabels, f(node))
	}

	return dsLabels
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
				"NET_ADMIN",
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

// volumes returns the volumes for the daemonset
func volumes() []corev1.Volume {
	pathTypeUnset := corev1.HostPathUnset

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
			Name: "opt-crowdstrike",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: "/opt/CrowdStrike",
					Type: &pathTypeUnset,
				},
			},
		},
	}
}

func DaemonsetConfigMapName(node *falconv1alpha1.FalconNodeSensor) string {
	if *node.Spec.Node.GKE.Enabled {
		return common.GKEAutoPilotConfigMapName
	}

	return node.Name + "-config"
}

// buildInitContainers returns the init containers for the daemonset
func buildInitContainers(image string, node *falconv1alpha1.FalconNodeSensor) []corev1.Container {
	privileged := true
	escalation := true
	runAsRoot := int64(0)

	initContainers := []corev1.Container{
		{
			Name:      "init-falconstore",
			Image:     image,
			Command:   common.FalconShellCommand,
			Args:      common.InitContainerArgs(),
			Resources: initContainerResources(node),
			SecurityContext: &corev1.SecurityContext{
				Privileged:               &privileged,
				RunAsUser:                &runAsRoot,
				ReadOnlyRootFilesystem:   isInitReadOnlyRootFilesystem(node),
				AllowPrivilegeEscalation: &escalation,
				Capabilities:             sensorCapabilities(node, true),
			},
			Env: []corev1.EnvVar{
				{
					Name: "POD_NODE_NAME",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "spec.nodeName",
						},
					},
				},
			},
		},
	}

	// Add init-configuration container if InitConfigImage is specified
	if node.Spec.Node.InitConfigImage != "" {
		initConfigContainer := corev1.Container{
			Name:    "init-configuration",
			Image:   node.Spec.Node.InitConfigImage,
			Command: common.FalconShellCommand,
			Args: []string{
				"-c",
				`# Check if AID file exists (indicating a restart)
if [ -f /opt/CrowdStrike/falconstore ]; then
    echo "AID detected, skipping configuration copy (restart scenario)"
else
    echo "No AID detected, copying configuration (first install)"
    # Add your configuration copy logic here
    # This is where the init container would copy pre-configuration
fi`,
			},
			SecurityContext: &corev1.SecurityContext{
				RunAsUser:                &runAsRoot,
				ReadOnlyRootFilesystem:   isInitReadOnlyRootFilesystem(node),
				AllowPrivilegeEscalation: &escalation,
			},
			VolumeMounts: []corev1.VolumeMount{
				{
					Name:      "opt-crowdstrike",
					MountPath: "/opt/CrowdStrike",
				},
			},
		}
		initContainers = append(initContainers, initConfigContainer)
	}

	return initContainers
}

func Daemonset(dsName, image, serviceAccount string, node *falconv1alpha1.FalconNodeSensor) *appsv1.DaemonSet {
	dsType := "daemonset"
	privileged := true
	escalation := true
	readOnlyFSDisabled := false
	hostpid := true
	hostnetwork := true
	hostipc := true
	runAsRoot := int64(0)
	dnsPolicy := corev1.DNSClusterFirstWithHostNet
	fsGroup := int64(nobodyGroup)
	podSecuityContext := corev1.PodSecurityContext{
		FSGroup: &fsGroup,
	}

	return &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dsName,
			Namespace: node.Spec.InstallNamespace,
			Labels:    common.CRLabels(dsType, dsName, common.FalconKernelSensor),
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: common.CRLabels(dsType, dsName, common.FalconKernelSensor),
			},
			UpdateStrategy: dsUpdateStrategy(node),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: dsManageAutoPilotLabels(dsType, dsName, dsAutoPilotDeployAllowlistLabel, node),
					Annotations: map[string]string{
						common.FalconContainerInjection: "disabled",
					},
				},
				Spec: corev1.PodSpec{
					// NodeSelector is set to linux until windows containers are supported for the Falcon sensor
					NodeSelector:                  common.NodeSelector,
					Affinity:                      nodeAffinity(node),
					Tolerations:                   *node.GetTolerations(),
					HostPID:                       hostpid,
					HostIPC:                       hostipc,
					HostNetwork:                   hostnetwork,
					DNSPolicy:                     dnsPolicy,
					TerminationGracePeriodSeconds: getTermGracePeriod(node),
					ImagePullSecrets:              pullSecrets(node),
					SecurityContext:               &podSecuityContext,
					InitContainers:                buildInitContainers(image, node),
					ServiceAccountName:            serviceAccount,
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
											Name: DaemonsetConfigMapName(node),
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
					Volumes:           volumes(),
					PriorityClassName: node.Spec.Node.PriorityClass.Name,
				},
			},
		},
	}
}

func RemoveNodeDirDaemonset(dsName, image, serviceAccount string, node *falconv1alpha1.FalconNodeSensor) *appsv1.DaemonSet {
	dsType := "cleanup"
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
			Namespace: node.Spec.InstallNamespace,
			Labels:    common.CRLabels(dsType, dsName, common.FalconKernelSensor),
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: common.CRLabels(dsType, dsName, common.FalconKernelSensor),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: dsManageAutoPilotLabels(dsType, dsName, dsAutoPilotCleanupAllowlistLabel, node),
					Annotations: map[string]string{
						common.FalconContainerInjection: "disabled",
					},
				},
				Spec: corev1.PodSpec{
					// NodeSelector is set to linux until windows containers are supported for the Falcon sensor
					NodeSelector:                  common.NodeSelector,
					Affinity:                      nodeAffinity(node),
					Tolerations:                   *node.GetTolerations(),
					HostPID:                       hostpid,
					TerminationGracePeriodSeconds: getTermGracePeriod(node),
					ImagePullSecrets:              pullSecrets(node),
					InitContainers: []corev1.Container{
						{
							Name:      "cleanup-opt-crowdstrike",
							Image:     image,
							Command:   common.FalconShellCommand,
							Args:      common.InitCleanupArgs(),
							Resources: initContainerResources(node),
							SecurityContext: &corev1.SecurityContext{
								Privileged:               &privileged,
								RunAsUser:                &runAsRoot,
								ReadOnlyRootFilesystem:   isInitReadOnlyRootFilesystem(node),
								AllowPrivilegeEscalation: &escalation,
								Capabilities:             sensorCapabilities(node, true),
							},
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
				},
			},
		},
	}
}
func isInitReadOnlyRootFilesystem(node *falconv1alpha1.FalconNodeSensor) *bool {
	disabled := node.Spec.Node.GKE.Enabled != nil && *node.Spec.Node.GKE.Enabled
	return &disabled
}
