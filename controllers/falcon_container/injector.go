package falcon

import (
	"context"
	"fmt"
	"reflect"

	"github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/crowdstrike/falcon-operator/pkg/tls"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	injectorName          = "injector"
	initContainerName     = "crowdstrike-falcon-init-container"
	injectorConfigMapName = "injector-config"
	injectorTLSSecretName = "injector-tls"
	falconVolumeName      = "crowdstrike-falcon-volume"
	falconVolumePath      = "/tmp/CrowdStrike"
)

var (
	FcLabels = map[string]string{
		common.FalconInstanceNameKey: injectorName,
		common.FalconInstanceKey:     "container_sensor",
		common.FalconComponentKey:    "container_sensor",
		common.FalconManagedByKey:    injectorName,
		common.FalconProviderKey:     common.FalconProviderValue,
		common.FalconPartOfKey:       "Falcon",
		common.FalconControllerKey:   "controller-manager",
	}
)

func (r *FalconContainerReconciler) reconcileInjectorTLSSecret(ctx context.Context, falconContainer *v1alpha1.FalconContainer) (*corev1.Secret, error) {
	existingInjectorTLSSecret := &corev1.Secret{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: injectorTLSSecretName, Namespace: r.Namespace()}, existingInjectorTLSSecret)
	if err != nil {
		if errors.IsNotFound(err) {
			validity := 3650
			if falconContainer.Spec.Injector.TLS.Validity != nil {
				validity = *falconContainer.Spec.Injector.TLS.Validity
			}
			c, k, b, err := tls.CertSetup(validity)
			if err != nil {
				return &corev1.Secret{}, fmt.Errorf("failed to generate Falcon Container PKI: %v", err)
			}
			injectorTLSSecret := r.newInjectorTLSSecret(c, k, b)
			if err = ctrl.SetControllerReference(falconContainer, injectorTLSSecret, r.Scheme); err != nil {
				return &corev1.Secret{}, fmt.Errorf("unable to set controller reference on injector TLS Secret%s: %v", injectorTLSSecret.ObjectMeta.Name, err)
			}
			return injectorTLSSecret, r.Create(ctx, falconContainer, injectorTLSSecret)
		}
		return &corev1.Secret{}, fmt.Errorf("unable to query existing injector TL secret %s: %v", injectorTLSSecretName, err)
	}
	return existingInjectorTLSSecret, nil

}

func (r *FalconContainerReconciler) newInjectorTLSSecret(c []byte, k []byte, b []byte) *corev1.Secret {
	return &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      injectorTLSSecretName,
			Namespace: r.Namespace(),
			Labels:    FcLabels,
		},
		Data: map[string][]byte{
			"tls.crt": c,
			"tls.key": k,
			"ca.crt":  b,
		},
		Type: corev1.SecretTypeOpaque,
	}
}

func (r *FalconContainerReconciler) reconcileDeployment(ctx context.Context, falconContainer *v1alpha1.FalconContainer) (*appsv1.Deployment, error) {
	imageUri, err := r.imageUri(ctx, falconContainer)
	if err != nil {
		return &appsv1.Deployment{}, fmt.Errorf("unable to determine falcon container image URI: %v", err)
	}
	deployment := r.newDeployment(imageUri, falconContainer)
	existingDeployment := &appsv1.Deployment{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: injectorName, Namespace: r.Namespace()}, existingDeployment)
	if err != nil {
		if errors.IsNotFound(err) {
			if err = ctrl.SetControllerReference(falconContainer, deployment, r.Scheme); err != nil {
				return &appsv1.Deployment{}, fmt.Errorf("unable to set controller reference on injector Deployment %s: %v", deployment.ObjectMeta.Name, err)
			}
			return deployment, r.Create(ctx, falconContainer, deployment)
		}
		return &appsv1.Deployment{}, fmt.Errorf("unable to query existing injector Deployment %s: %v", injectorName, err)
	}
	// Selectors are immutable
	if !reflect.DeepEqual(deployment.Spec.Selector, existingDeployment.Spec.Selector) {
		// TODO: Handle reconciling label selectors
		return &appsv1.Deployment{}, fmt.Errorf("unable to reconcile deployment; label selectors are not equal but are immutable")
	} else if !reflect.DeepEqual(deployment.Spec.Template, existingDeployment.Spec.Template) {
		existingDeployment.Spec.Template = deployment.Spec.Template
		return existingDeployment, r.Update(ctx, falconContainer, existingDeployment)
	}
	return existingDeployment, nil

}

func (r *FalconContainerReconciler) newDeployment(imageUri string, falconContainer *v1alpha1.FalconContainer) *appsv1.Deployment {
	imagePullSecrets := []corev1.LocalObjectReference{{Name: common.FalconPullSecretName}}
	azureVolumeName := "azure-config"
	azureVolumePath := "/run/azure.json"
	hostPathFile := corev1.HostPathFile
	if common.FalconPullSecretName != falconContainer.Spec.Injector.ImagePullSecretName {
		imagePullSecrets = append(imagePullSecrets, corev1.LocalObjectReference{Name: falconContainer.Spec.Injector.ImagePullSecretName})
	}
	resources := &corev1.ResourceRequirements{}
	if falconContainer.Spec.Injector.Resources != nil {
		resources = falconContainer.Spec.Injector.Resources
	}
	var replicas int32 = 1
	var rootUid int64 = 0
	runNonRoot := true
	initRunAsNonRoot := false
	volumes := []corev1.Volume{
		{
			Name: injectorTLSSecretName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName: injectorTLSSecretName,
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
	initContainers := []corev1.Container{}
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
	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: appsv1.SchemeGroupVersion.String(),
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      injectorName,
			Namespace: r.Namespace(),
			Labels:    FcLabels,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: FcLabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: FcLabels,
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
					ImagePullSecrets: imagePullSecrets,
					SecurityContext: &corev1.PodSecurityContext{
						RunAsNonRoot: &runNonRoot,
					},
					InitContainers: initContainers,
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
								},
							},
							VolumeMounts: []corev1.VolumeMount{
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
							},
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

func (r *FalconContainerReconciler) injectorPodReady(ctx context.Context, falconContainer *v1alpha1.FalconContainer) (*corev1.Pod, error) {
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(r.Namespace()),
		client.MatchingLabels(FcLabels),
	}
	if err := r.Client.List(ctx, podList, listOpts...); err != nil {
		return nil, fmt.Errorf("unable to list pods: %v", err)
	}
	for _, pod := range podList.Items {
		for _, cond := range pod.Status.Conditions {
			if cond.Type == corev1.PodReady && cond.Status == corev1.ConditionTrue {
				return &pod, nil
			}
		}
	}
	return &corev1.Pod{}, fmt.Errorf("No Injector pod found in a Ready state")
}
