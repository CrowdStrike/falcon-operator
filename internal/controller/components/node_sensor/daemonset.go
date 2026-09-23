package node_sensor

import (
	"context"
	"fmt"
	"maps"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	k8sutils "github.com/crowdstrike/falcon-operator/internal/controller/common"
	pkgcommon "github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/operator-framework/operator-lib/proxy"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/util/retry"
)

// daemonSet builds the DaemonSet for the FalconClusterGuard node sensor.
func (n *NodeSensor) daemonSet() *appsv1.DaemonSet {
	namespace := n.cfg.InstallNamespace
	nodeSpec := n.cfg.NodeSensor

	dsLabels := pkgcommon.CRLabels("daemonset", n.prefix()+"-sensor", pkgcommon.ClusterGuardNodeSensorComponentName)
	privileged := true
	runAsUser := int64(0)
	hostPathType := corev1.HostPathUnset

	imageUri := n.cfg.Image
	if imageUri == "" && n.cfg.FalconAPI != nil && n.cfg.FalconAPI.CID != nil {
		imageUri = *n.cfg.FalconAPI.CID
	}

	imagePullPolicy := n.cfg.ImagePullPolicy
	if imagePullPolicy == "" {
		imagePullPolicy = corev1.PullIfNotPresent
	}

	terminationGracePeriod := nodeSpec.TerminationGracePeriod
	if terminationGracePeriod == 0 {
		terminationGracePeriod = 60
	}

	tolerations := []corev1.Toleration{}
	if nodeSpec.Tolerations != nil {
		tolerations = *nodeSpec.Tolerations
	}

	updateStrategy := appsv1.DaemonSetUpdateStrategy{Type: appsv1.RollingUpdateDaemonSetStrategyType}
	if nodeSpec.DSUpdateStrategy.Type == appsv1.RollingUpdateDaemonSetStrategyType || nodeSpec.DSUpdateStrategy.Type == "" {
		rollingUpdateSettings := appsv1.RollingUpdateDaemonSet{}
		if nodeSpec.DSUpdateStrategy.RollingUpdate.MaxSurge != nil {
			rollingUpdateSettings.MaxSurge = nodeSpec.DSUpdateStrategy.RollingUpdate.MaxSurge
		}
		if nodeSpec.DSUpdateStrategy.RollingUpdate.MaxUnavailable != nil {
			rollingUpdateSettings.MaxUnavailable = nodeSpec.DSUpdateStrategy.RollingUpdate.MaxUnavailable
		}
		updateStrategy = appsv1.DaemonSetUpdateStrategy{
			Type:          appsv1.RollingUpdateDaemonSetStrategyType,
			RollingUpdate: &rollingUpdateSettings,
		}
	} else if nodeSpec.DSUpdateStrategy.Type == appsv1.OnDeleteDaemonSetStrategyType {
		updateStrategy = appsv1.DaemonSetUpdateStrategy{Type: appsv1.OnDeleteDaemonSetStrategyType}
	}

	containerResources := n.dsResources()

	apiServiceName := pkgcommon.ClusterGuardAPIServiceName + "." + namespace + ".svc"

	secretDefaultMode := int32(420)
	podSpec := corev1.PodSpec{
		ServiceAccountName:            pkgcommon.ClusterGuardSensorServiceAccountName,
		TerminationGracePeriodSeconds: &terminationGracePeriod,
		PriorityClassName:             n.cfg.NodeSensor.PriorityClass.Name,
		HostNetwork:                   true,
		DNSPolicy:                     corev1.DNSClusterFirstWithHostNet,
		HostPID:                       true,
		HostIPC:                       true,
		NodeSelector: map[string]string{
			"kubernetes.io/os": "linux",
		},
		SecurityContext: &corev1.PodSecurityContext{
			FSGroup: func() *int64 { g := int64(65534); return &g }(),
		},
		Tolerations: tolerations,
		InitContainers: []corev1.Container{
			{
				Name:            "init-falconstore",
				Image:           imageUri,
				ImagePullPolicy: imagePullPolicy,
				Command:         []string{"/bin/bash"},
				Args:            pkgcommon.InitContainerArgs(),
				Resources:       n.initContainerResources(),
				SecurityContext: &corev1.SecurityContext{
					RunAsUser:                &runAsUser,
					Privileged:               &privileged,
					AllowPrivilegeEscalation: &privileged,
					ReadOnlyRootFilesystem:   n.isInitReadOnlyRootFilesystem(),
					Capabilities:             n.sensorCapabilities(true),
				},
				Env: []corev1.EnvVar{
					{Name: "POD_NODE_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{APIVersion: "v1", FieldPath: "spec.nodeName"}}},
					{Name: "API_SERVICE_NAME", Value: apiServiceName},
				},
				VolumeMounts: []corev1.VolumeMount{
					{Name: "cs-config", MountPath: pkgcommon.FalconConfigDir},
					{Name: "falconstore", MountPath: pkgcommon.FalconStoreFile},
					{Name: "falcon-node-sensor-tls-certs", MountPath: "/run/secrets/tls", ReadOnly: true},
					{Name: "falcon-api-ca", MountPath: "/run/secrets/ca", ReadOnly: true},
				},
			},
		},
		Containers: []corev1.Container{
			{
				Name:            "falcon-node-sensor",
				Image:           imageUri,
				ImagePullPolicy: imagePullPolicy,
				SecurityContext: &corev1.SecurityContext{
					RunAsUser:                &runAsUser,
					Privileged:               &privileged,
					ReadOnlyRootFilesystem:   func() *bool { b := false; return &b }(),
					AllowPrivilegeEscalation: &privileged,
					Capabilities:             n.sensorCapabilities(false),
				},
				Resources: containerResources,
				Env: append([]corev1.EnvVar{
					{Name: "POD_NODE_NAME", ValueFrom: &corev1.EnvVarSource{FieldRef: &corev1.ObjectFieldSelector{APIVersion: "v1", FieldPath: "spec.nodeName"}}},
					{Name: "API_SERVICE_NAME", Value: apiServiceName},
				}, pkgcommon.OperatorMetaEnvVars()...),
				EnvFrom: []corev1.EnvFromSource{
					{ConfigMapRef: &corev1.ConfigMapEnvSource{LocalObjectReference: corev1.LocalObjectReference{Name: n.configMapName()}}},
				},
				VolumeMounts: []corev1.VolumeMount{
					{Name: "cs-config", MountPath: pkgcommon.FalconConfigDir},
					{Name: "falconstore", MountPath: pkgcommon.FalconStoreFile},
					{Name: "falcon-node-sensor-tls-certs", MountPath: "/run/secrets/tls", ReadOnly: true},
					{Name: "falcon-api-ca", MountPath: "/run/secrets/ca", ReadOnly: true},
				},
			},
		},
		Volumes: []corev1.Volume{
			{Name: "cs-config", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: pkgcommon.FalconConfigDir, Type: &hostPathType}}},
			{Name: "falconstore", VolumeSource: corev1.VolumeSource{HostPath: &corev1.HostPathVolumeSource{Path: pkgcommon.FalconStoreFile, Type: &hostPathType}}},
			{Name: "falcon-node-sensor-tls-certs", VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: "falcon-node-sensor-tls", DefaultMode: &secretDefaultMode}}},
			{Name: "falcon-api-ca", VolumeSource: corev1.VolumeSource{Secret: &corev1.SecretVolumeSource{SecretName: pkgcommon.ClusterGuardAPICASecretName, DefaultMode: &secretDefaultMode}}},
		},
	}

	if nodeSpec.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution != nil ||
		len(nodeSpec.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution) > 0 {
		podSpec.Affinity = &corev1.Affinity{NodeAffinity: &nodeSpec.NodeAffinity}
	}

	return &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      n.prefix() + "-sensor",
			Namespace: namespace,
			Labels:    dsLabels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector:       &metav1.LabelSelector{MatchLabels: map[string]string{"app": n.prefix() + "-sensor"}},
			UpdateStrategy: updateStrategy,
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: n.dsManageAutoPilotLabels("daemonset", n.prefix()+"-sensor", pkgcommon.ClusterGuardNodeSensorComponentName, n.dsAutoPilotDeployAllowlistLabel),
					Annotations: map[string]string{
						pkgcommon.FalconContainerInjection: "disabled",
						"checksum/config":                  pkgcommon.ConfigMapChecksum(n.configMap()),
					},
				},
				Spec: podSpec,
			},
		},
	}
}

// cleanupDaemonSet builds the cleanup DaemonSet that removes
// /opt/CrowdStrike from each node during FalconClusterGuard finalization.
func (n *NodeSensor) cleanupDaemonSet() *appsv1.DaemonSet {
	namespace := n.cfg.InstallNamespace
	imageUri := n.cfg.Image
	imagePullPolicy := n.cfg.ImagePullPolicy

	dsLabels := pkgcommon.CRLabels("daemonset", n.prefix()+"-sensor-cleanup", pkgcommon.ClusterGuardComponentName)

	if imagePullPolicy == "" {
		imagePullPolicy = corev1.PullIfNotPresent
	}
	privileged := true
	runAsUser := int64(0)
	terminationGracePeriod := n.cfg.NodeSensor.TerminationGracePeriod
	if terminationGracePeriod == 0 {
		terminationGracePeriod = 60
	}
	readOnlyRootFilesystem := true
	allowPrivilegeEscalation := true
	disallowPrivilegeEscalation := false

	return &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "DaemonSet",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      n.prefix() + "-sensor-cleanup",
			Namespace: namespace,
			Labels:    dsLabels,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{"app": n.prefix() + "-sensor-cleanup"},
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: n.dsManageAutoPilotLabels("daemonset", n.prefix()+"-sensor-cleanup", pkgcommon.ClusterGuardComponentName, n.dsAutoPilotCleanupAllowlistLabel),
					Annotations: map[string]string{
						pkgcommon.FalconContainerInjection: "disabled",
					},
				},
				Spec: corev1.PodSpec{
					ServiceAccountName:            pkgcommon.ClusterGuardSensorCleanupServiceAccountName,
					TerminationGracePeriodSeconds: &terminationGracePeriod,
					HostPID:                       true,
					NodeSelector: map[string]string{
						"kubernetes.io/os": "linux",
					},
					InitContainers: []corev1.Container{
						{
							Name:            "cleanup-opt-crowdstrike",
							Image:           imageUri,
							ImagePullPolicy: imagePullPolicy,
							Command:         []string{"/bin/bash"},
							Args:            pkgcommon.InitCleanupArgs(),
							SecurityContext: &corev1.SecurityContext{
								RunAsUser:                &runAsUser,
								Privileged:               &privileged,
								AllowPrivilegeEscalation: &allowPrivilegeEscalation,
								ReadOnlyRootFilesystem:   func() *bool { b := false; return &b }(),
							},
						},
					},
					Containers: []corev1.Container{
						{
							Name:            "cleanup-sleep",
							Image:           imageUri,
							ImagePullPolicy: imagePullPolicy,
							Command:         []string{"/bin/bash"},
							Args:            pkgcommon.CleanupSleep(),
							SecurityContext: &corev1.SecurityContext{
								Privileged:               &disallowPrivilegeEscalation,
								ReadOnlyRootFilesystem:   &readOnlyRootFilesystem,
								AllowPrivilegeEscalation: &allowPrivilegeEscalation,
							},
						},
					},
				},
			},
		},
	}
}

// sensorCapabilities returns the capabilities for the sensor containers based on GKE configuration.
func (n *NodeSensor) sensorCapabilities(initContainer bool) *corev1.Capabilities {
	if n.cfg.NodeSensor.GKE.Enabled != nil && *n.cfg.NodeSensor.GKE.Enabled {
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

// dsAutoPilotDeployAllowlistLabel returns GKE Autopilot allowlist labels for deployment DaemonSet
func (n *NodeSensor) dsAutoPilotDeployAllowlistLabel() map[string]string {
	if n.cfg.NodeSensor.GKE.Enabled != nil && *n.cfg.NodeSensor.GKE.Enabled && n.cfg.NodeSensor.GKE.DeployAllowListVersion != nil {
		return map[string]string{
			pkgcommon.GKEAutoPilotAllowListLabelKey: fmt.Sprintf("%s-%s", pkgcommon.GKEAutoPilotDeployDSAllowlistPrefix, *n.cfg.NodeSensor.GKE.DeployAllowListVersion),
		}
	}
	return nil
}

// dsAutoPilotCleanupAllowlistLabel returns GKE Autopilot allowlist labels for cleanup DaemonSet
func (n *NodeSensor) dsAutoPilotCleanupAllowlistLabel() map[string]string {
	if n.cfg.NodeSensor.GKE.Enabled != nil && *n.cfg.NodeSensor.GKE.Enabled && n.cfg.NodeSensor.GKE.CleanupAllowListVersion != nil {
		return map[string]string{
			pkgcommon.GKEAutoPilotAllowListLabelKey: fmt.Sprintf("%s-%s", pkgcommon.GKEAutoPilotCleanupAllowlistPrefix, *n.cfg.NodeSensor.GKE.CleanupAllowListVersion),
		}
	}
	return nil
}

// dsManageAutoPilotLabels merges autopilot labels with CR labels
func (n *NodeSensor) dsManageAutoPilotLabels(dsType string, dsName string, componentName string, labelFunc func() map[string]string) map[string]string {
	dsLabels := pkgcommon.CRLabels(dsType, dsName, componentName)
	// Add the "app" label for DaemonSet selector matching
	dsLabels["app"] = dsName
	autoPilotLabel := labelFunc()
	if autoPilotLabel != nil {
		maps.Copy(dsLabels, autoPilotLabel)
	}
	return dsLabels
}

// initContainerResources returns resource requirements for init containers
func (n *NodeSensor) initContainerResources() corev1.ResourceRequirements {
	if n.cfg.NodeSensor.Backend == "bpf" && (n.cfg.NodeSensor.SensorResources != falconv1alpha1.FalconClusterGuardResources{} || (n.cfg.NodeSensor.GKE.Enabled != nil && *n.cfg.NodeSensor.GKE.Enabled)) {
		return corev1.ResourceRequirements{
			Limits: corev1.ResourceList{
				corev1.ResourceCPU:              resource.MustParse("10m"),
				corev1.ResourceEphemeralStorage: resource.MustParse("100Mi"),
				corev1.ResourceMemory:           resource.MustParse("50Mi"),
			},
			Requests: corev1.ResourceList{
				corev1.ResourceCPU:              resource.MustParse("10m"),
				corev1.ResourceEphemeralStorage: resource.MustParse("100Mi"),
				corev1.ResourceMemory:           resource.MustParse("50Mi"),
			},
		}
	}
	return corev1.ResourceRequirements{}
}

// dsResources returns resource requirements for the main container
func (n *NodeSensor) dsResources() corev1.ResourceRequirements {
	nodeSpec := n.cfg.NodeSensor
	if nodeSpec.Backend == "bpf" {
		limitResources := corev1.ResourceList{}
		requestsResources := corev1.ResourceList{}

		// Set GKE defaults if enabled
		if nodeSpec.GKE.Enabled != nil && *nodeSpec.GKE.Enabled {
			limitResources = corev1.ResourceList{
				corev1.ResourceCPU:              resource.MustParse("750m"),
				corev1.ResourceMemory:           resource.MustParse("1.5Gi"),
				corev1.ResourceEphemeralStorage: resource.MustParse("100Mi"),
			}
			requestsResources = corev1.ResourceList{
				corev1.ResourceCPU:              resource.MustParse("750m"),
				corev1.ResourceMemory:           resource.MustParse("1.5Gi"),
				corev1.ResourceEphemeralStorage: resource.MustParse("100Mi"),
			}
		}

		// Override with user-specified values
		if nodeSpec.SensorResources.Limits.CPU != "" {
			limitResources[corev1.ResourceCPU] = resource.MustParse(nodeSpec.SensorResources.Limits.CPU)
		}
		if nodeSpec.SensorResources.Limits.Memory != "" {
			limitResources[corev1.ResourceMemory] = resource.MustParse(nodeSpec.SensorResources.Limits.Memory)
		}
		if nodeSpec.SensorResources.Limits.EphemeralStorage != "" {
			limitResources[corev1.ResourceEphemeralStorage] = resource.MustParse(nodeSpec.SensorResources.Limits.EphemeralStorage)
		}
		if nodeSpec.SensorResources.Requests.CPU != "" {
			requestsResources[corev1.ResourceCPU] = resource.MustParse(nodeSpec.SensorResources.Requests.CPU)
		}
		if nodeSpec.SensorResources.Requests.Memory != "" {
			requestsResources[corev1.ResourceMemory] = resource.MustParse(nodeSpec.SensorResources.Requests.Memory)
		}
		if nodeSpec.SensorResources.Requests.EphemeralStorage != "" {
			requestsResources[corev1.ResourceEphemeralStorage] = resource.MustParse(nodeSpec.SensorResources.Requests.EphemeralStorage)
		}

		return corev1.ResourceRequirements{
			Limits:   limitResources,
			Requests: requestsResources,
		}
	}
	return corev1.ResourceRequirements{}
}

// configMapName returns the ConfigMap name based on GKE settings.
// For GKE autopilot, the WorkloadAllowlist requires the exact name.
// Otherwise, the name is derived from the prefix.
func (n *NodeSensor) configMapName() string {
	if n.cfg.NodeSensor.GKE.Enabled != nil && *n.cfg.NodeSensor.GKE.Enabled {
		return pkgcommon.GKEAutoPilotConfigMapName
	}
	return n.prefix() + "-sensor-config"
}

// isInitReadOnlyRootFilesystem returns whether init container should have read-only root filesystem
func (n *NodeSensor) isInitReadOnlyRootFilesystem() *bool {
	readOnly := false
	return &readOnly
}

// BuildResourceRequirements converts a falconv1alpha1.Resources spec into corev1.ResourceRequirements.
func BuildResourceRequirements(res falconv1alpha1.FalconClusterGuardResources) corev1.ResourceRequirements {
	reqs := corev1.ResourceRequirements{
		Limits:   corev1.ResourceList{},
		Requests: corev1.ResourceList{},
	}
	if res.Limits.Memory != "" {
		reqs.Limits[corev1.ResourceMemory] = resource.MustParse(res.Limits.Memory)
	}
	if res.Limits.CPU != "" {
		reqs.Limits[corev1.ResourceCPU] = resource.MustParse(res.Limits.CPU)
	}
	if res.Requests.Memory != "" {
		reqs.Requests[corev1.ResourceMemory] = resource.MustParse(res.Requests.Memory)
	}
	if res.Requests.CPU != "" {
		reqs.Requests[corev1.ResourceCPU] = resource.MustParse(res.Requests.CPU)
	}
	return reqs
}

func (n *NodeSensor) reconcileDaemonSet(ctx context.Context) error {
	ds := n.daemonSet()

	// Apply Kubernetes-equivalent defaults for RollingUpdate fields that are left nil
	// when the user does not configure them, so the desired spec matches the live object.
	if ds.Spec.UpdateStrategy.RollingUpdate != nil {
		ru := ds.Spec.UpdateStrategy.RollingUpdate
		if ru.MaxUnavailable == nil {
			v := intstr.FromInt32(1)
			ru.MaxUnavailable = &v
		}
		if ru.MaxSurge == nil {
			v := intstr.FromInt32(0)
			ru.MaxSurge = &v
		}
	}

	// Inject operator proxy env vars into the desired spec containers before create/update.
	if len(proxy.ReadProxyVarsFromEnv()) > 0 {
		for i, container := range ds.Spec.Template.Spec.Containers {
			ds.Spec.Template.Spec.Containers[i].Env = append(container.Env, proxy.ReadProxyVarsFromEnv()...)
		}
	}

	existing := &appsv1.DaemonSet{}
	found, err := k8sutils.GetOrCreate(ctx, n.r, n.cfg.Request, n.cfg.Owner, n.cfg.Status, ds, existing,
		types.NamespacedName{Name: n.prefix() + "-sensor", Namespace: n.cfg.InstallNamespace},
		"Failed to get FalconClusterGuard sensor DaemonSet")
	if !found || err != nil {
		return err
	}

	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if err := pkgcommon.GetNamespacedObject(ctx, n.r, n.r.GetK8sReader(),
			types.NamespacedName{Name: n.prefix() + "-sensor", Namespace: n.cfg.InstallNamespace},
			existing); err != nil {
			return err
		}

		updated := false

		// UpdateStrategy
		if !equality.Semantic.DeepEqual(ds.Spec.UpdateStrategy, existing.Spec.UpdateStrategy) {
			n.r.GetLog().V(1).Info("Updating FalconClusterGuard sensor DaemonSet: UpdateStrategy changed",
				"old", existing.Spec.UpdateStrategy,
				"new", ds.Spec.UpdateStrategy)
			existing.Spec.UpdateStrategy = ds.Spec.UpdateStrategy
			updated = true
		}

		// Volumes
		if !equality.Semantic.DeepEqual(ds.Spec.Template.Spec.Volumes, existing.Spec.Template.Spec.Volumes) {
			n.r.GetLog().V(1).Info("Updating FalconClusterGuard sensor DaemonSet: Volumes changed")
			existing.Spec.Template.Spec.Volumes = ds.Spec.Template.Spec.Volumes
			updated = true
		}

		// NodeAffinity
		if ds.Spec.Template.Spec.Affinity != nil {
			if existing.Spec.Template.Spec.Affinity == nil {
				existing.Spec.Template.Spec.Affinity = &corev1.Affinity{}
			}
			if !equality.Semantic.DeepEqual(ds.Spec.Template.Spec.Affinity.NodeAffinity, existing.Spec.Template.Spec.Affinity.NodeAffinity) {
				n.r.GetLog().V(1).Info("Updating FalconClusterGuard sensor DaemonSet: NodeAffinity changed",
					"old", existing.Spec.Template.Spec.Affinity.NodeAffinity,
					"new", ds.Spec.Template.Spec.Affinity.NodeAffinity)
				existing.Spec.Template.Spec.Affinity.NodeAffinity = ds.Spec.Template.Spec.Affinity.NodeAffinity
				updated = true
			}
		}

		// Tolerations (merge: spec is authoritative; preserve existing tolerations not in spec)
		mergedTolerations := ds.Spec.Template.Spec.Tolerations
		for _, existingTol := range existing.Spec.Template.Spec.Tolerations {
			found := false
			for _, specTol := range ds.Spec.Template.Spec.Tolerations {
				if existingTol.Key == specTol.Key && existingTol.Effect == specTol.Effect {
					found = true
					break
				}
			}
			if !found {
				mergedTolerations = append(mergedTolerations, existingTol)
			}
		}
		if !equality.Semantic.DeepEqual(existing.Spec.Template.Spec.Tolerations, mergedTolerations) {
			n.r.GetLog().V(1).Info("Updating FalconClusterGuard sensor DaemonSet: Tolerations changed",
				"old", existing.Spec.Template.Spec.Tolerations,
				"new", mergedTolerations)
			existing.Spec.Template.Spec.Tolerations = mergedTolerations
			updated = true
		}

		// TerminationGracePeriodSeconds
		if !equality.Semantic.DeepEqual(ds.Spec.Template.Spec.TerminationGracePeriodSeconds, existing.Spec.Template.Spec.TerminationGracePeriodSeconds) {
			n.r.GetLog().V(1).Info("Updating FalconClusterGuard sensor DaemonSet: TerminationGracePeriodSeconds changed",
				"old", existing.Spec.Template.Spec.TerminationGracePeriodSeconds,
				"new", ds.Spec.Template.Spec.TerminationGracePeriodSeconds)
			existing.Spec.Template.Spec.TerminationGracePeriodSeconds = ds.Spec.Template.Spec.TerminationGracePeriodSeconds
			updated = true
		}

		// Pod template labels (GKE Autopilot allowlist labels and standard CR labels)
		if !equality.Semantic.DeepEqual(ds.Spec.Template.Labels, existing.Spec.Template.Labels) {
			n.r.GetLog().V(1).Info("Updating FalconClusterGuard sensor DaemonSet: pod template labels changed")
			existing.Spec.Template.Labels = ds.Spec.Template.Labels
			updated = true
		}

		// Pod template annotations (includes checksum/config for ConfigMap-triggered rerollout)
		if !equality.Semantic.DeepEqual(ds.Spec.Template.Annotations, existing.Spec.Template.Annotations) {
			n.r.GetLog().V(1).Info("Updating FalconClusterGuard sensor DaemonSet: pod template annotations changed")
			existing.Spec.Template.Annotations = ds.Spec.Template.Annotations
			updated = true
		}

		// InitContainers: per-field checks
		if len(ds.Spec.Template.Spec.InitContainers) > 0 && len(existing.Spec.Template.Spec.InitContainers) > 0 {
			specInit := ds.Spec.Template.Spec.InitContainers[0]
			existingInit := &existing.Spec.Template.Spec.InitContainers[0]
			if existingInit.Image != specInit.Image {
				n.r.GetLog().V(1).Info("Updating FalconClusterGuard sensor DaemonSet: InitContainer image changed",
					"old", existingInit.Image, "new", specInit.Image)
				existingInit.Image = specInit.Image
				updated = true
			}
			if !equality.Semantic.DeepEqual(existingInit.VolumeMounts, specInit.VolumeMounts) {
				n.r.GetLog().V(1).Info("Updating FalconClusterGuard sensor DaemonSet: InitContainer VolumeMounts changed")
				existingInit.VolumeMounts = specInit.VolumeMounts
				updated = true
			}
			if !equality.Semantic.DeepEqual(existingInit.Env, specInit.Env) {
				n.r.GetLog().V(1).Info("Updating FalconClusterGuard sensor DaemonSet: InitContainer Env changed")
				existingInit.Env = specInit.Env
				updated = true
			}
			if !equality.Semantic.DeepEqual(existingInit.Args, specInit.Args) {
				n.r.GetLog().V(1).Info("Updating FalconClusterGuard sensor DaemonSet: InitContainer Args changed")
				existingInit.Args = specInit.Args
				updated = true
			}
			if existingInit.SecurityContext != nil && specInit.SecurityContext != nil &&
				!equality.Semantic.DeepEqual(existingInit.SecurityContext.Capabilities, specInit.SecurityContext.Capabilities) {
				n.r.GetLog().V(1).Info("Updating FalconClusterGuard sensor DaemonSet: InitContainer Capabilities changed")
				existingInit.SecurityContext.Capabilities = specInit.SecurityContext.Capabilities
				updated = true
			}
			if !equality.Semantic.DeepEqual(existingInit.Resources, specInit.Resources) {
				n.r.GetLog().V(1).Info("Updating FalconClusterGuard sensor DaemonSet: InitContainer Resources changed",
					"old", existingInit.Resources, "new", specInit.Resources)
				existingInit.Resources = specInit.Resources
				updated = true
			}
		} else if !equality.Semantic.DeepEqual(ds.Spec.Template.Spec.InitContainers, existing.Spec.Template.Spec.InitContainers) {
			n.r.GetLog().V(1).Info("Updating FalconClusterGuard sensor DaemonSet: InitContainers changed")
			existing.Spec.Template.Spec.InitContainers = ds.Spec.Template.Spec.InitContainers
			updated = true
		}

		// Containers: per-field checks
		if len(ds.Spec.Template.Spec.Containers) > 0 && len(existing.Spec.Template.Spec.Containers) > 0 {
			specC := ds.Spec.Template.Spec.Containers[0]
			existingC := &existing.Spec.Template.Spec.Containers[0]
			if existingC.Image != specC.Image {
				n.r.GetLog().V(1).Info("Updating FalconClusterGuard sensor DaemonSet: Container image changed",
					"old", existingC.Image, "new", specC.Image)
				existingC.Image = specC.Image
				updated = true
			}
			if !equality.Semantic.DeepEqual(existingC.VolumeMounts, specC.VolumeMounts) {
				n.r.GetLog().V(1).Info("Updating FalconClusterGuard sensor DaemonSet: Container VolumeMounts changed")
				existingC.VolumeMounts = specC.VolumeMounts
				updated = true
			}
			if !equality.Semantic.DeepEqual(existingC.Resources, specC.Resources) {
				n.r.GetLog().V(1).Info("Updating FalconClusterGuard sensor DaemonSet: Container Resources changed",
					"old", existingC.Resources, "new", specC.Resources)
				existingC.Resources = specC.Resources
				updated = true
			}
			if !equality.Semantic.DeepEqual(existingC.Env, specC.Env) {
				// Merge existing proxy env vars from the cluster into the spec env before comparing,
				// to avoid overwriting proxy vars that were injected by the operator environment.
				mergedEnv := pkgcommon.MergeEnvVars(specC.Env, existingC.Env, pkgcommon.ProxyEnvNamesWithLowerCase())
				if !equality.Semantic.DeepEqual(existingC.Env, mergedEnv) {
					n.r.GetLog().V(1).Info("Updating FalconClusterGuard sensor DaemonSet: Container Env changed")
					existingC.Env = mergedEnv
					updated = true
				}
			}
			if existingC.SecurityContext != nil && specC.SecurityContext != nil &&
				!equality.Semantic.DeepEqual(existingC.SecurityContext.Capabilities, specC.SecurityContext.Capabilities) {
				n.r.GetLog().V(1).Info("Updating FalconClusterGuard sensor DaemonSet: Container Capabilities changed")
				existingC.SecurityContext.Capabilities = specC.SecurityContext.Capabilities
				updated = true
			}
			if !equality.Semantic.DeepEqual(existingC.EnvFrom, specC.EnvFrom) {
				n.r.GetLog().V(1).Info("Updating FalconClusterGuard sensor DaemonSet: Container EnvFrom changed")
				existingC.EnvFrom = specC.EnvFrom
				updated = true
			}
		} else if !equality.Semantic.DeepEqual(ds.Spec.Template.Spec.Containers, existing.Spec.Template.Spec.Containers) {
			n.r.GetLog().V(1).Info("Updating FalconClusterGuard sensor DaemonSet: Containers changed")
			existing.Spec.Template.Spec.Containers = ds.Spec.Template.Spec.Containers
			updated = true
		}

		// Reconcile proxy env vars: append any new proxy vars from the operator environment,
		// and update the values of any existing proxy vars that have changed.
		if len(proxy.ReadProxyVarsFromEnv()) > 0 {
			for i, container := range existing.Spec.Template.Spec.Containers {
				newEnv := pkgcommon.AppendUniqueEnvVars(container.Env, proxy.ReadProxyVarsFromEnv())
				updatedEnv := pkgcommon.UpdateEnvVars(container.Env, proxy.ReadProxyVarsFromEnv())
				if !equality.Semantic.DeepEqual(existing.Spec.Template.Spec.Containers[i].Env, newEnv) {
					existing.Spec.Template.Spec.Containers[i].Env = newEnv
					updated = true
				}
				if !equality.Semantic.DeepEqual(existing.Spec.Template.Spec.Containers[i].Env, updatedEnv) {
					existing.Spec.Template.Spec.Containers[i].Env = updatedEnv
					updated = true
				}
			}
			if updated {
				n.r.GetLog().V(1).Info("Updating FalconClusterGuard sensor DaemonSet: Proxy env vars changed")
			}
		}

		if updated {
			existing.SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind("DaemonSet"))
			return k8sutils.Update(n.r, ctx, n.cfg.Request, n.r.GetLog(), n.cfg.Owner, n.cfg.Status, existing)
		}
		return nil
	})
	if err != nil {
		n.r.GetLog().Error(err, "Failed to update FalconClusterGuard sensor DaemonSet after retries")
		return err
	}
	return nil
}

func (n *NodeSensor) reconcileCleanupServiceAccount(ctx context.Context) error {
	sa := n.cleanupServiceAccount()
	_, err := k8sutils.GetOrCreate(ctx, n.r, n.cfg.Request, n.cfg.Owner, n.cfg.Status, sa, &corev1.ServiceAccount{},
		types.NamespacedName{Name: pkgcommon.ClusterGuardSensorCleanupServiceAccountName, Namespace: n.cfg.InstallNamespace},
		"Failed to get FalconClusterGuard sensor cleanup ServiceAccount")
	return err
}

func (n *NodeSensor) reconcileCleanupDaemonSet(ctx context.Context) error {
	ds := n.cleanupDaemonSet()
	existing := &appsv1.DaemonSet{}
	found, err := k8sutils.GetOrCreate(ctx, n.r, n.cfg.Request, n.cfg.Owner, n.cfg.Status, ds, existing,
		types.NamespacedName{Name: n.prefix() + "-sensor-cleanup", Namespace: n.cfg.InstallNamespace},
		"Failed to get FalconClusterGuard sensor cleanup DaemonSet")
	if !found || err != nil {
		return err
	}

	updated := false

	// InitContainers: per-field checks
	if len(ds.Spec.Template.Spec.InitContainers) > 0 && len(existing.Spec.Template.Spec.InitContainers) > 0 {
		specInit := ds.Spec.Template.Spec.InitContainers[0]
		existingInit := &existing.Spec.Template.Spec.InitContainers[0]
		if existingInit.Image != specInit.Image {
			n.r.GetLog().V(1).Info("Updating FalconClusterGuard cleanup DaemonSet: InitContainer image changed",
				"old", existingInit.Image, "new", specInit.Image)
			existingInit.Image = specInit.Image
			updated = true
		}
		if !equality.Semantic.DeepEqual(existingInit.Args, specInit.Args) {
			n.r.GetLog().V(1).Info("Updating FalconClusterGuard cleanup DaemonSet: InitContainer Args changed")
			existingInit.Args = specInit.Args
			updated = true
		}
		if existingInit.SecurityContext != nil && specInit.SecurityContext != nil &&
			!equality.Semantic.DeepEqual(existingInit.SecurityContext.Capabilities, specInit.SecurityContext.Capabilities) {
			n.r.GetLog().V(1).Info("Updating FalconClusterGuard cleanup DaemonSet: InitContainer Capabilities changed")
			existingInit.SecurityContext.Capabilities = specInit.SecurityContext.Capabilities
			updated = true
		}
	} else if !equality.Semantic.DeepEqual(ds.Spec.Template.Spec.InitContainers, existing.Spec.Template.Spec.InitContainers) {
		n.r.GetLog().V(1).Info("Updating FalconClusterGuard cleanup DaemonSet: InitContainers changed")
		existing.Spec.Template.Spec.InitContainers = ds.Spec.Template.Spec.InitContainers
		updated = true
	}

	// Containers: per-field checks
	if len(ds.Spec.Template.Spec.Containers) > 0 && len(existing.Spec.Template.Spec.Containers) > 0 {
		specC := ds.Spec.Template.Spec.Containers[0]
		existingC := &existing.Spec.Template.Spec.Containers[0]
		if existingC.Image != specC.Image {
			n.r.GetLog().V(1).Info("Updating FalconClusterGuard cleanup DaemonSet: Container image changed",
				"old", existingC.Image, "new", specC.Image)
			existingC.Image = specC.Image
			updated = true
		}
		if existingC.SecurityContext != nil && specC.SecurityContext != nil &&
			!equality.Semantic.DeepEqual(existingC.SecurityContext.Capabilities, specC.SecurityContext.Capabilities) {
			n.r.GetLog().V(1).Info("Updating FalconClusterGuard cleanup DaemonSet: Container Capabilities changed")
			existingC.SecurityContext.Capabilities = specC.SecurityContext.Capabilities
			updated = true
		}
	} else if !equality.Semantic.DeepEqual(ds.Spec.Template.Spec.Containers, existing.Spec.Template.Spec.Containers) {
		n.r.GetLog().V(1).Info("Updating FalconClusterGuard cleanup DaemonSet: Containers changed")
		existing.Spec.Template.Spec.Containers = ds.Spec.Template.Spec.Containers
		updated = true
	}

	if updated {
		existing.SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind("DaemonSet"))
		return k8sutils.Update(n.r, ctx, n.cfg.Request, n.r.GetLog(), n.cfg.Owner, n.cfg.Status, existing)
	}
	return nil
}
