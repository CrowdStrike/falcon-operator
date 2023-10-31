package assets

import (
	"testing"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/google/go-cmp/cmp"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// TestDeployment tests the Deployment function
func TestSideCarDeployment(t *testing.T) {
	falconContainer := &falconv1alpha1.FalconContainer{}
	falconContainer.Spec.Injector.Resources = &corev1.ResourceRequirements{}
	falconContainer.Spec.Injector.AzureConfigPath = "/test"
	falconContainer.Spec.Registry.TLS.CACertificateConfigMap = "test"
	falconContainer.Spec.Registry.TLS.CACertificate = "test"
	port := int32(123)
	falconContainer.Spec.Injector.ListenPort = &port
	falconContainer.Spec.Injector.Replicas = &port
	want := testSideCarDeployment("test", "test", "test", "test", falconContainer)

	got := SideCarDeployment("test", "test", "test", "test", falconContainer)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Deployment() mismatch (-want +got): %s", diff)
	}
}

// TestAdmissionDeployment tests the Admission Controller Deployment function
func TestAdmissionDeployment(t *testing.T) {
	falconAdmission := &falconv1alpha1.FalconAdmission{}
	falconAdmission.Spec.AdmissionConfig.ResourcesClient = &corev1.ResourceRequirements{}
	falconAdmission.Spec.AdmissionConfig.ResourcesAC = &corev1.ResourceRequirements{}
	port := int32(123)
	falconAdmission.Spec.AdmissionConfig.Port = &port
	falconAdmission.Spec.AdmissionConfig.Replicas = &port
	falconAdmission.Spec.AdmissionConfig.ContainerPort = &port
	want := testAdmissionDeployment("test", "test", "test", "test", falconAdmission)

	got := AdmissionDeployment("test", "test", "test", "test", falconAdmission)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Deployment() mismatch (-want +got): %s", diff)
	}
}

// TestAdmissionDepUpdateStrategy tests the Admission Controller Deployment Update Strategy function
func TestAdmissionDepUpdateStrategy(t *testing.T) {
	falconAdmission := falconv1alpha1.FalconAdmission{}

	// Test RollingUpdate return value
	want := appsv1.DeploymentStrategy{
		Type: appsv1.RollingUpdateDeploymentStrategyType,
		RollingUpdate: &appsv1.RollingUpdateDeployment{
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

	falconAdmission.Spec.AdmissionConfig.DepUpdateStrategy.RollingUpdate.MaxUnavailable = &intstr.IntOrString{Type: intstr.Int, IntVal: 1}
	falconAdmission.Spec.AdmissionConfig.DepUpdateStrategy.RollingUpdate.MaxSurge = &intstr.IntOrString{Type: intstr.Int, IntVal: 1}

	got := admissionDepUpdateStrategy(&falconAdmission)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("admissionDepUpdateStrategy() mismatch (-want +got): %s", diff)
	}
}

// testSideCarDeployment is a helper function to create a Deployment object for testing
func testSideCarDeployment(name string, namespace string, component string, imageUri string, falconContainer *falconv1alpha1.FalconContainer) *appsv1.Deployment {
	replicas := int32(123)
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
			Name:      "test",
			Namespace: "test",
			Labels:    common.CRLabels("deployment", "test", "test"),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: common.CRLabels("deployment", "test", "test"),
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
					TopologySpreadConstraints: []corev1.TopologySpreadConstraint{{
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

// testAdmissionDeployment is a helper function to create a Deployment object for testing
func testAdmissionDeployment(name string, namespace string, component string, imageUri string, falconAdmission *falconv1alpha1.FalconAdmission) *appsv1.Deployment {
	runNonRoot := true
	readOnlyRootFilesystem := true
	allowPrivilegeEscalation := false
	shareProcessNamespace := true
	resourcesClient := &corev1.ResourceRequirements{}
	resourcesAC := &corev1.ResourceRequirements{}
	sizeLimitTmp := resource.MustParse("256Mi")
	sizeLimitPrivate := resource.MustParse("4Ki")
	labels := common.CRLabels("deployment", name, component)

	if falconAdmission.Spec.AdmissionConfig.ResourcesClient != nil {
		resourcesClient = falconAdmission.Spec.AdmissionConfig.ResourcesClient
	}

	if falconAdmission.Spec.AdmissionConfig.ResourcesAC != nil {
		resourcesAC = falconAdmission.Spec.AdmissionConfig.ResourcesAC
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
												Key:      "node-role.kubernetes.io/master",
												Operator: corev1.NodeSelectorOpDoesNotExist,
											},
										},
									},
								},
							},
						},
					},
					TopologySpreadConstraints: []corev1.TopologySpreadConstraint{{
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
									ContainerPort: *falconAdmission.Spec.AdmissionConfig.Port,
									Name:          common.FalconAdmissionServiceHTTPSName,
									Protocol:      corev1.ProtocolTCP,
								},
							},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "crowdstrike-falcon-vol0",
									MountPath: "/tmp",
								},
								{
									Name:      "crowdstrike-falcon-vol1",
									MountPath: "/var/private",
								},
								{
									Name:      name + "-tls-certs",
									MountPath: "/run/secrets/tls",
									ReadOnly:  true,
								},
							},
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
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "crowdstrike-falcon-vol0",
									MountPath: "/tmp",
								},
								{
									Name:      "crowdstrike-falcon-vol1",
									MountPath: "/var/private",
								},
							},
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
					Volumes: []corev1.Volume{
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
					},
				},
			},
		},
	}
}
