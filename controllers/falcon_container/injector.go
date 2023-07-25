package falcon

import (
	"context"
	"fmt"
	"reflect"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/internal/controller/assets"
	"github.com/go-logr/logr"
	"github.com/operator-framework/operator-lib/proxy"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/crowdstrike/falcon-operator/pkg/tls"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/errors"
	types "k8s.io/apimachinery/pkg/types"
)

const (
	injectorName                  = "falcon-sidecar-injector"
	injectorTLSSecretName         = "falcon-sidecar-injector-tls"
	injectorConfigMapName         = "falcon-sidecar-injector-config"
	registryCABundleConfigMapName = "falcon-sidecar-registry-certs"
)

func (r *FalconContainerReconciler) reconcileInjectorTLSSecret(ctx context.Context, log logr.Logger, falconContainer *falconv1alpha1.FalconContainer) (*corev1.Secret, error) {
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
			secretData := map[string][]byte{
				"tls.crt": c,
				"tls.key": k,
				"ca.crt":  b,
			}
			injectorTLSSecret := assets.Secret(injectorTLSSecretName, r.Namespace(), common.FalconSidecarSensor, secretData, corev1.SecretTypeTLS)
			if err = ctrl.SetControllerReference(falconContainer, injectorTLSSecret, r.Scheme); err != nil {
				return &corev1.Secret{}, fmt.Errorf("unable to set controller reference on injector TLS Secret%s: %v", injectorTLSSecret.ObjectMeta.Name, err)
			}
			return injectorTLSSecret, r.Create(ctx, log, falconContainer, injectorTLSSecret)
		}
		return &corev1.Secret{}, fmt.Errorf("unable to query existing injector TL secret %s: %v", injectorTLSSecretName, err)
	}
	return existingInjectorTLSSecret, nil

}

func (r *FalconContainerReconciler) reconcileDeployment(ctx context.Context, log logr.Logger, falconContainer *falconv1alpha1.FalconContainer) (*appsv1.Deployment, error) {
	update := false

	imageUri, err := r.imageUri(ctx, falconContainer)
	if err != nil {
		return &appsv1.Deployment{}, fmt.Errorf("unable to determine falcon container image URI: %v", err)
	}

	deployment := assets.SideCarDeployment(injectorName, r.Namespace(), common.FalconSidecarSensor, imageUri, falconContainer)
	existingDeployment := &appsv1.Deployment{}

	if len(proxy.ReadProxyVarsFromEnv()) > 0 {
		for i, container := range deployment.Spec.Template.Spec.Containers {
			deployment.Spec.Template.Spec.Containers[i].Env = append(container.Env, proxy.ReadProxyVarsFromEnv()...)
		}
	}

	err = r.Client.Get(ctx, types.NamespacedName{Name: injectorName, Namespace: r.Namespace()}, existingDeployment)
	if err != nil {
		if errors.IsNotFound(err) {
			if err = ctrl.SetControllerReference(falconContainer, deployment, r.Scheme); err != nil {
				return &appsv1.Deployment{}, fmt.Errorf("unable to set controller reference on injector Deployment %s: %v", deployment.ObjectMeta.Name, err)
			}
			return deployment, r.Create(ctx, log, falconContainer, deployment)
		}
		return &appsv1.Deployment{}, fmt.Errorf("unable to query existing injector Deployment %s: %v", injectorName, err)
	}

	// Selectors are immutable
	if !reflect.DeepEqual(deployment.Spec.Selector, existingDeployment.Spec.Selector) {
		// TODO: Handle reconciling label selectors
		return &appsv1.Deployment{}, fmt.Errorf("unable to reconcile deployment; label selectors are not equal but are immutable")
	}

	if len(proxy.ReadProxyVarsFromEnv()) > 0 {
		for i, container := range existingDeployment.Spec.Template.Spec.Containers {
			newContainerEnv := common.AppendUniqueEnvVars(container.Env, proxy.ReadProxyVarsFromEnv())
			updatedContainerEnv := common.UpdateEnvVars(container.Env, proxy.ReadProxyVarsFromEnv())
			if !equality.Semantic.DeepEqual(existingDeployment.Spec.Template.Spec.Containers[i].Env, newContainerEnv) {
				existingDeployment.Spec.Template.Spec.Containers[i].Env = newContainerEnv
				update = true
			}
			if !equality.Semantic.DeepEqual(existingDeployment.Spec.Template.Spec.Containers[i].Env, updatedContainerEnv) {
				existingDeployment.Spec.Template.Spec.Containers[i].Env = updatedContainerEnv
				update = true
			}
			if update {
				log.Info("Updating FalconNodeSensor Deployment Proxy Settings")
			}
		}

	}

	if !reflect.DeepEqual(deployment.Spec.Template.Spec.Containers[0].Image, existingDeployment.Spec.Template.Spec.Containers[0].Image) {
		existingDeployment.Spec.Template.Spec.Containers[0].Image = deployment.Spec.Template.Spec.Containers[0].Image
		update = true
	}

	if !reflect.DeepEqual(deployment.Spec.Template.Spec.Containers[0].Ports, existingDeployment.Spec.Template.Spec.Containers[0].Ports) {
		existingDeployment.Spec.Template.Spec.Containers[0].Ports = deployment.Spec.Template.Spec.Containers[0].Ports
		update = true
	}

	if !reflect.DeepEqual(deployment.Spec.Template.Spec.Containers[0].ImagePullPolicy, existingDeployment.Spec.Template.Spec.Containers[0].ImagePullPolicy) {
		existingDeployment.Spec.Template.Spec.Containers[0].ImagePullPolicy = deployment.Spec.Template.Spec.Containers[0].ImagePullPolicy
		update = true
	}

	if !reflect.DeepEqual(deployment.Spec.Template.Spec.Containers[0].Resources, existingDeployment.Spec.Template.Spec.Containers[0].Resources) {
		existingDeployment.Spec.Template.Spec.Containers[0].Resources = deployment.Spec.Template.Spec.Containers[0].Resources
		update = true
	}

	if !reflect.DeepEqual(deployment.Spec.Replicas, existingDeployment.Spec.Replicas) {
		existingDeployment.Spec.Replicas = deployment.Spec.Replicas
		update = true
	}

	if !reflect.DeepEqual(deployment.Spec.Template.Spec.TopologySpreadConstraints, existingDeployment.Spec.Template.Spec.TopologySpreadConstraints) {
		existingDeployment.Spec.Template.Spec.TopologySpreadConstraints = deployment.Spec.Template.Spec.TopologySpreadConstraints
		update = true
	}

	if update {
		return existingDeployment, r.Update(ctx, log, falconContainer, existingDeployment)
	}

	return existingDeployment, nil

}

func (r *FalconContainerReconciler) injectorPodReady(ctx context.Context, falconContainer *falconv1alpha1.FalconContainer) (*corev1.Pod, error) {
	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(r.Namespace()),
		client.MatchingLabels{common.FalconComponentKey: common.FalconSidecarSensor},
	}

	if err := r.List(ctx, podList, listOpts...); err != nil {
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
