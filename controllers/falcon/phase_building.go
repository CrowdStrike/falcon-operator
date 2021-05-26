package falcon

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/falcon_container"
)

func (r *FalconConfigReconciler) phaseBuildingReconcile(ctx context.Context, instance *falconv1alpha1.FalconConfig, logger logr.Logger) (ctrl.Result, error) {
	logger.Info("Phase: Building")

	err := r.ensureDockercfg(ctx, instance.ObjectMeta.Namespace)
	if err != nil {
		return r.error(ctx, instance, "Cannot find dockercfg secret from the current namespace", err)
	}
	imageStream, err := r.imageStream(ctx, instance.ObjectMeta.Namespace)
	if err != nil {
		return r.error(ctx, instance, "Cannot access image stream", err)
	}
	logger.Info(imageStream.Status.DockerImageRepository)

	err = r.refreshContainerImage(ctx, instance, imageStream.Status.DockerImageRepository)
	if err != nil {
		return ctrl.Result{}, fmt.Errorf("Error when reconciling Falcon Container Image: %w", err)
	}

	instance.Status.Phase = falconv1alpha1.PhaseDone

	err = r.Client.Status().Update(ctx, instance)
	return ctrl.Result{}, err
}

func (r *FalconConfigReconciler) refreshContainerImage(ctx context.Context, falconConfig *falconv1alpha1.FalconConfig, destination string) error {
	image := falcon_container.NewImageRefresher(ctx, r.Log, falconConfig.Spec.FalconAPI.ApiConfig())
	return image.Refresh(destination)
}

func (r *FalconConfigReconciler) ensureDockercfg(ctx context.Context, namespace string) error {
	dockercfg, err := r.getDockercfg(ctx, namespace)
	if err != nil {
		return err
	}
	return ioutil.WriteFile("/tmp/.dockercfg", dockercfg, 0600)
}

func (r *FalconConfigReconciler) getDockercfg(ctx context.Context, namespace string) ([]byte, error) {
	secrets := &corev1.SecretList{}
	err := r.Client.List(ctx, secrets, client.InNamespace(namespace))
	if err != nil {
		return []byte{}, err
	}

	for _, secret := range secrets.Items {
		if secret.Data == nil {
			continue
		}
		if secret.Type != "kubernetes.io/dockercfg" {
			continue
		}

		if secret.ObjectMeta.Annotations == nil || secret.ObjectMeta.Annotations["kubernetes.io/service-account.name"] != "builder" {
			continue
		}

		value, ok := secret.Data[".dockercfg"]
		if !ok {
			continue
		}
		return value, nil
	}

	return []byte{}, fmt.Errorf("Cannot find suitable secret in namespace %s to push falcon-image to the registry", namespace)
}

func (r *FalconConfigReconciler) error(ctx context.Context, instance *falconv1alpha1.FalconConfig, message string, err error) (ctrl.Result, error) {
	userError := fmt.Errorf("%s %w", message, err)

	return ctrl.Result{}, userError

}
