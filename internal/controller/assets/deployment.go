package assets

import (
	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// SideCarDeployment returns a Deployment object for the CrowdStrike Falcon sidecar
func SideCarDeployment(name string, namespace string, component string, imageUri string, falconContainer *falconv1alpha1.FalconContainer) *appsv1.Deployment {
	initContainerName := "crowdstrike-falcon-init-container"
	injectorConfigMapName := "falcon-sidecar-injector-config"
	registryCABundleConfigMapName := "falcon-sidecar-registry-certs"
	injectorTLSSecretName := "falcon-sidecar-injector-tls"
	falconVolumeName := "crowdstrike-falcon-volume"
	falconVolumePath := "/tmp/CrowdStrike"
	azureVolumeName := "azure-config"
	azureVolumePath := "/run/azure.json"
	certPath := "/etc/docker/certs.d/falcon-system-certs"
	hostPathFile := corev1.HostPathFile
	resources := &corev1.ResourceRequirements{}
	var rootUid int64 = 0
	var readMode int32 = 420
	runNonRoot := true
	initRunAsNonRoot := false
	initContainers := []corev1.Container{}
	var registryCAConfigMapName string = ""
	labels := common.CRLabels("deployment", name, component)

	if falconContainer.Spec.Injector.Resources != nil {
		resources = falconContainer.Spec.Injector.Resources
	}

	volumes := []corev1.Volume{
		{
			Name: injectorTLSSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  injectorTLSSecretName,
					DefaultMode: &readMode,
				},
			},
		},
		{
			Name: falconVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}

	volumeMounts := []corev1.VolumeMount{
		{
			Name:      injectorTLSSecretName,
			MountPath: "/run/secrets/tls",
			ReadOnly:  true,
		},
		{
			Name:      falconVolumeName,
			MountPath: falconVolumePath,
			ReadOnly:  true,
		},
	}

	if falconContainer.Spec.Injector.AzureConfigPath != "" {
		initContainers = append(initContainers, corev1.Container{
			Name:            initContainerName,
			ImagePullPolicy: falconContainer.Spec.Injector.ImagePullPolicy,
			Image:           imageUri,
			Command: []string{
				"bash",
				"-c",
				"cp /run/azure.json /tmp/CrowdStrike/; chmod a+r /tmp/CrowdStrike/azure.json",
			},
			SecurityContext: &corev1.SecurityContext{
				RunAsUser:    &rootUid,
				RunAsNonRoot: &initRunAsNonRoot,
				Privileged:   &initRunAsNonRoot,
			},
			VolumeMounts: []corev1.VolumeMount{{
				Name:      falconVolumeName,
				MountPath: falconVolumePath,
			}, {
				Name:      azureVolumeName,
				MountPath: azureVolumePath,
				ReadOnly:  true,
			}},
		})

		volumes = append(volumes, corev1.Volume{
			Name: azureVolumeName,
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: falconContainer.Spec.Injector.AzureConfigPath,
					Type: &hostPathFile,
				},
			}})
	}

	if falconContainer.Spec.Registry.TLS.CACertificateConfigMap != "" {
		registryCAConfigMapName = falconContainer.Spec.Registry.TLS.CACertificateConfigMap
	}

	if falconContainer.Spec.Registry.TLS.CACertificate != "" {
		registryCAConfigMapName = registryCABundleConfigMapName
	}

	if registryCAConfigMapName != "" {
		volumes = append(volumes, corev1.Volume{
			Name: registryCAConfigMapName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: registryCAConfigMapName,
					},
				},
			},
		})
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      registryCAConfigMapName,
			ReadOnly:  true,
			MountPath: certPath,
		})
	}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: falconContainer.Spec.Injector.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
					Annotations: map[string]string{
						common.FalconContainerInjection: "disabled",
					},
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
												Key:      "kubernetes.io/arch",
												Operator: corev1.NodeSelectorOpIn,
												Values:   []string{"amd64"},
											},
										},
									},
								},
							},
						},
					},
					TopologySpreadConstraints: []corev1.TopologySpreadConstraint{
						{
							MaxSkew:           1,
							TopologyKey:       "kubernetes.io/hostname",
							WhenUnsatisfiable: corev1.ScheduleAnyway,
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{common.FalconInstanceNameKey: name},
							},
						},
					},
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: &runNonRoot,
					},
					InitContainers:     initContainers,
					ServiceAccountName: common.SidecarServiceAccountName,
					Containers: []corev1.Container{
						{
							Name:            "falcon-sensor",
							Image:           imageUri,
							ImagePullPolicy: falconContainer.Spec.Injector.ImagePullPolicy,
							Command:         common.FalconInjectorCommand,
							EnvFrom: []corev1.EnvFromSource{
								{
									ConfigMapRef: &corev1.ConfigMapEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: injectorConfigMapName,
										},
									},
								},
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: *falconContainer.Spec.Injector.ListenPort,
									Name:          common.FalconServiceHTTPSName,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							VolumeMounts: volumeMounts,
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   common.FalconContainerProbePath,
										Port:   intstr.IntOrString{IntVal: *falconContainer.Spec.Injector.ListenPort},
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
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   common.FalconContainerProbePath,
										Port:   intstr.IntOrString{IntVal: *falconContainer.Spec.Injector.ListenPort},
										Scheme: corev1.URISchemeHTTPS,
									},
								},
								InitialDelaySeconds: 5,
								TimeoutSeconds:      1,
								PeriodSeconds:       10,
								SuccessThreshold:    1,
								FailureThreshold:    3,
							},
							Resources: *resources,
						},
					},
					Volumes: volumes,
				},
			},
		},
	}
}

// ImageDeployment returns a Deployment object for the CrowdStrike Falcon IAR Controller
func ImageDeployment(name string, namespace string, component string, imageUri string, falconImage *falconv1alpha1.FalconImage) *appsv1.Deployment {
	azureConfig := "/etc/kubernetes/azure.json"
	labels := common.CRLabels("deployment", name, component)
	var replicaCount int32 = 1
	initContainers := []corev1.Container{}
	chartName := "falcon-image-analyzer"
	hostPathFile := corev1.HostPathFile
	var rootUid int64 = 0
	initRunAsNonRoot := false
	privileged := false

	initContainers = append(initContainers, corev1.Container{
		Name:            chartName + "-init-container",
		ImagePullPolicy: falconImage.Spec.ImageConfig.ImagePullPolicy,
		Image:           imageUri,
		Command: []string{
			"bash",
			"-c",
			"curl -sS -H 'Metadata-Flavor: Google' 'http://metadata.google.internal/computeMetadata/v1/instance/service-accounts/default/token' --retry 30 --retry-connrefused --retry-max-time 60 --connect-timeout 3 --fail --retry-all-errors > /dev/null && exit 0 || echo 'Retry limit exceeded. Failed to wait for metadata server to be available. Check if the gke-metadata-server Pod in the kube-system namespace is healthy.' >&2; exit 1",
		},
		SecurityContext: &corev1.SecurityContext{
			RunAsUser:    &rootUid,
			RunAsNonRoot: &initRunAsNonRoot,
			Privileged:   &privileged,
		},
	},
	)

	volumes := []corev1.Volume{
		{
			Name: "azure-config",
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: azureConfig,
					Type: &hostPathFile,
				},
			},
		},
	}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicaCount,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
					Annotations: map[string]string{
						common.FalconContainerInjection: "disabled",
					},
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
					TopologySpreadConstraints: []corev1.TopologySpreadConstraint{
						{
							MaxSkew:           1,
							TopologyKey:       "kubernetes.io/hostname",
							WhenUnsatisfiable: corev1.ScheduleAnyway,
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{common.FalconInstanceNameKey: name},
							},
						},
					},
					InitContainers: initContainers,
					Containers: []corev1.Container{
						{
							Name: "falcon-client",
							SecurityContext: &corev1.SecurityContext{
								RunAsUser:  &rootUid,
								Privileged: &privileged,
							},
							Resources:       corev1.ResourceRequirements{},
							Image:           imageUri,
							ImagePullPolicy: falconImage.Spec.ImageConfig.ImagePullPolicy,
							Args:            []string{"-runmode", "watcher"},
							EnvFrom: []corev1.EnvFromSource{
								{
									ConfigMapRef: &corev1.ConfigMapEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: name,
										},
									},
								},
							},
							VolumeMounts: []corev1.VolumeMount{},
						},
					},
					ServiceAccountName: common.ImageServiceAccountName,
					SecurityContext:    &corev1.PodSecurityContext{},
					NodeSelector:       common.NodeSelector,
					Volumes:            volumes,
					//Tolerations:        falconImage.Spec.ImageConfig.Tolerations,
					//PriorityClassName:  falconImage.Spec.ImageConfig.PriorityClass.Name,
				},
			},
		},
	}
}

// AdmissionDeployment returns a Deployment object for the CrowdStrike Falcon Admission Controller
func AdmissionDeployment(name string, namespace string, component string, imageUri string, falconAdmission *falconv1alpha1.FalconAdmission) *appsv1.Deployment {
	runNonRoot := true
	readOnlyRootFilesystem := true
	allowPrivilegeEscalation := false
	shareProcessNamespace := true
	resourcesClient := &corev1.ResourceRequirements{}
	resourcesAC := &corev1.ResourceRequirements{}
	sizeLimitTmp := resource.MustParse("256Mi")
	sizeLimitPrivate := resource.MustParse("4Ki")
	labels := common.CRLabels("deployment", name, component)
	registryCAConfigMapName := ""
	registryCABundleConfigMapName := name + "-registry-certs"

	if falconAdmission.Spec.AdmissionConfig.ResourcesClient != nil {
		resourcesClient = falconAdmission.Spec.AdmissionConfig.ResourcesClient
	}

	if falconAdmission.Spec.AdmissionConfig.ResourcesAC != nil {
		resourcesAC = falconAdmission.Spec.AdmissionConfig.ResourcesAC
	}

	volumes := []corev1.Volume{
		{
			Name: name + "-tls-certs",
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: name + "-tls",
				},
			},
		},
		{
			Name: "crowdstrike-falcon-vol0",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					SizeLimit: &sizeLimitTmp,
				},
			},
		},
		{
			Name: "crowdstrike-falcon-vol1",
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					SizeLimit: &sizeLimitPrivate,
				},
			},
		},
	}

	if falconAdmission.Spec.Registry.TLS.CACertificateConfigMap != "" {
		registryCAConfigMapName = falconAdmission.Spec.Registry.TLS.CACertificateConfigMap
	}

	if falconAdmission.Spec.Registry.TLS.CACertificate != "" {
		registryCAConfigMapName = registryCABundleConfigMapName
	}

	if registryCAConfigMapName != "" {
		volumes = append(volumes, corev1.Volume{
			Name: registryCAConfigMapName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: registryCAConfigMapName,
					},
				},
			},
		})
	}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: falconAdmission.Spec.AdmissionConfig.Replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: labels,
			},
			Strategy: admissionDepUpdateStrategy(falconAdmission),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: labels,
					Annotations: map[string]string{
						common.FalconContainerInjection: "disabled",
					},
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
												Key:      "kubernetes.io/arch",
												Operator: corev1.NodeSelectorOpIn,
												Values:   []string{"amd64"},
											},
										},
									},
								},
							},
						},
					},
					TopologySpreadConstraints: []corev1.TopologySpreadConstraint{
						{
							MaxSkew:           1,
							TopologyKey:       "kubernetes.io/hostname",
							WhenUnsatisfiable: corev1.ScheduleAnyway,
							LabelSelector: &metav1.LabelSelector{
								MatchLabels: map[string]string{common.FalconInstanceNameKey: name},
							},
						},
					},
					ShareProcessNamespace: &shareProcessNamespace,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: &runNonRoot,
						SeccompProfile: &corev1.SeccompProfile{
							Type: corev1.SeccompProfileTypeRuntimeDefault,
						},
					},
					ServiceAccountName: common.AdmissionServiceAccountName,
					NodeSelector:       common.NodeSelector,
					PriorityClassName:  common.FalconPriorityClassName,
					Containers: []corev1.Container{
						{
							Name:            "falcon-client",
							Image:           imageUri,
							ImagePullPolicy: falconAdmission.Spec.AdmissionConfig.ImagePullPolicy,
							Args:            []string{"client"},
							SecurityContext: &corev1.SecurityContext{
								ReadOnlyRootFilesystem:   &readOnlyRootFilesystem,
								AllowPrivilegeEscalation: &allowPrivilegeEscalation,
								RunAsNonRoot:             &runNonRoot,
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{
										"ALL",
									},
								},
							},
							Env: []corev1.EnvVar{
								{
									Name: "__CS_POD_NAMESPACE",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											APIVersion: "v1",
											FieldPath:  "metadata.namespace",
										},
									},
								},
								{
									Name: "__CS_POD_NAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											APIVersion: "v1",
											FieldPath:  "metadata.name",
										},
									},
								},
								{
									Name: "__CS_POD_NODENAME",
									ValueFrom: &corev1.EnvVarSource{
										FieldRef: &corev1.ObjectFieldSelector{
											APIVersion: "v1",
											FieldPath:  "spec.nodeName",
										},
									},
								},
							},
							EnvFrom: []corev1.EnvFromSource{
								{
									ConfigMapRef: &corev1.ConfigMapEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: name + "-config",
										},
									},
								},
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: *falconAdmission.Spec.AdmissionConfig.ContainerPort,
									Name:          common.FalconServiceHTTPSName,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							VolumeMounts: admissionDepVolumeMounts(name, registryCAConfigMapName, true),
							StartupProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   common.FalconAdmissionClientStartupProbePath,
										Port:   intstr.IntOrString{IntVal: *falconAdmission.Spec.AdmissionConfig.ContainerPort},
										Scheme: corev1.URISchemeHTTPS,
									},
								},
								InitialDelaySeconds: 5,
								TimeoutSeconds:      1,
								PeriodSeconds:       2,
								SuccessThreshold:    1,
								FailureThreshold:    30,
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   common.FalconAdmissionClientLivenessProbePath,
										Port:   intstr.IntOrString{IntVal: *falconAdmission.Spec.AdmissionConfig.ContainerPort},
										Scheme: corev1.URISchemeHTTPS,
									},
								},
								InitialDelaySeconds: 5,
								TimeoutSeconds:      1,
								PeriodSeconds:       10,
								SuccessThreshold:    1,
								FailureThreshold:    3,
							},
							Resources: *resourcesClient,
						},
						{
							Name:            "falcon-kac",
							Image:           imageUri,
							ImagePullPolicy: falconAdmission.Spec.AdmissionConfig.ImagePullPolicy,

							SecurityContext: &corev1.SecurityContext{
								ReadOnlyRootFilesystem:   &readOnlyRootFilesystem,
								AllowPrivilegeEscalation: &allowPrivilegeEscalation,
								RunAsNonRoot:             &runNonRoot,
								Capabilities: &corev1.Capabilities{
									Drop: []corev1.Capability{
										"ALL",
									},
								},
							},
							EnvFrom: []corev1.EnvFromSource{
								{
									ConfigMapRef: &corev1.ConfigMapEnvSource{
										LocalObjectReference: corev1.LocalObjectReference{
											Name: name + "-config",
										},
									},
								},
							},
							VolumeMounts: admissionDepVolumeMounts(name, registryCAConfigMapName, false),
							StartupProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   common.FalconAdmissionStartupProbePath,
										Port:   intstr.IntOrString{IntVal: *falconAdmission.Spec.AdmissionConfig.ContainerPort},
										Scheme: corev1.URISchemeHTTPS,
									},
								},
								InitialDelaySeconds: 5,
								TimeoutSeconds:      1,
								PeriodSeconds:       2,
								SuccessThreshold:    1,
								FailureThreshold:    30,
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Path:   common.FalconAdmissionLivenessProbePath,
										Port:   intstr.IntOrString{IntVal: *falconAdmission.Spec.AdmissionConfig.ContainerPort},
										Scheme: corev1.URISchemeHTTPS,
									},
								},
								InitialDelaySeconds: 5,
								TimeoutSeconds:      1,
								PeriodSeconds:       10,
								SuccessThreshold:    1,
								FailureThreshold:    3,
							},
							Resources: *resourcesAC,
						},
					},
					Volumes: volumes,
				},
			},
		},
	}
}

func admissionDepVolumeMounts(name string, registryCAConfigMapName string, client bool) []corev1.VolumeMount {
	certPath := "/etc/docker/certs.d/falcon-admission-certs"

	volumeMounts := []corev1.VolumeMount{
		{
			Name:      "crowdstrike-falcon-vol0",
			MountPath: "/tmp",
		},
		{
			Name:      "crowdstrike-falcon-vol1",
			MountPath: "/var/private",
		},
	}

	if client {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      name + "-tls-certs",
			MountPath: "/run/secrets/tls",
			ReadOnly:  true,
		})
	}

	if registryCAConfigMapName != "" {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      registryCAConfigMapName,
			ReadOnly:  true,
			MountPath: certPath,
		})
	}

	return volumeMounts
}

func admissionDepUpdateStrategy(admission *falconv1alpha1.FalconAdmission) appsv1.DeploymentStrategy {
	rollingUpdateSettings := appsv1.RollingUpdateDeployment{}

	if admission.Spec.AdmissionConfig.DepUpdateStrategy.RollingUpdate.MaxSurge != nil {
		rollingUpdateSettings.MaxSurge = admission.Spec.AdmissionConfig.DepUpdateStrategy.RollingUpdate.MaxSurge
	}

	if admission.Spec.AdmissionConfig.DepUpdateStrategy.RollingUpdate.MaxUnavailable != nil {
		rollingUpdateSettings.MaxUnavailable = admission.Spec.AdmissionConfig.DepUpdateStrategy.RollingUpdate.MaxUnavailable
	}

	return appsv1.DeploymentStrategy{
		Type:          appsv1.RollingUpdateDeploymentStrategyType,
		RollingUpdate: &rollingUpdateSettings,
	}
}
