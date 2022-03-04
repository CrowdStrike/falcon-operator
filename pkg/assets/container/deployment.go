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

func ContainerDeployment(dsName string, nsName string, falconcontainer *falconv1alpha1.FalconContainer) *appsv1.Deployment {
	return containerDeployment(dsName, nsName, falconcontainer)
}

func containerDeployment(dsName string, nsName string, falconcontainer *falconv1alpha1.FalconContainer) *appsv1.Deployment {
	var replicas int32 = 1
	runNonRoot := true
	var injectorPort int32 = 4433
	recpu := "10m"
	remem := "20Mi"

	labels := map[string]string{
		common.FalconInstanceNameKey: dsName,
		common.FalconInstanceKey:     "container_sensor",
		common.FalconComponentKey:    "container_sensor",
		common.FalconManagedByKey:    dsName,
		common.FalconProviderKey:     common.FalconProviderValue,
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
					Containers: []corev1.Container{
						{
							Name:            "falcon-sensor",
							Image:           "falcon-sensor",
							ImagePullPolicy: "Always",
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
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      dsName + "-tls-certs",
									MountPath: "/run/secrets/tls",
									ReadOnly:  true,
								},
							},
							ReadinessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
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
								ProbeHandler: corev1.ProbeHandler{
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
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse(recpu),
									corev1.ResourceMemory: resource.MustParse(remem),
								},
							},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: dsName + "-tls-certs",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: dsName + "-tls",
								},
							},
						},
					},
				},
			},
		},
	}
}
