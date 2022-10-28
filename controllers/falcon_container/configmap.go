package falcon

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/crowdstrike/falcon-operator/pkg/falcon_api"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *FalconContainerReconciler) reconcileConfigMap(ctx context.Context, falconContainer *v1alpha1.FalconContainer) (*corev1.ConfigMap, error) {
	configMap, err := r.newConfigMap(ctx, falconContainer)
	if err != nil {
		return configMap, fmt.Errorf("unable to render expected configmap: %v", err)
	}
	existingConfigMap := &corev1.ConfigMap{}
	err = r.Client.Get(ctx, types.NamespacedName{Name: injectorName, Namespace: r.Namespace()}, existingConfigMap)
	if err != nil {
		if errors.IsNotFound(err) {
			if err = ctrl.SetControllerReference(falconContainer, configMap, r.Scheme); err != nil {
				return &corev1.ConfigMap{}, fmt.Errorf("unable to set controller reference on config map %s: %v", configMap.ObjectMeta.Name, err)
			}
			return configMap, r.Create(ctx, falconContainer, configMap)
		}
		return &corev1.ConfigMap{}, fmt.Errorf("unable to query existing config map %s: %v", injectorName, err)
	}
	if reflect.DeepEqual(configMap.Data, existingConfigMap.Data) {
		return existingConfigMap, nil
	}
	existingConfigMap.Data = configMap.Data
	return existingConfigMap, r.Update(ctx, falconContainer, existingConfigMap)

}

func (r *FalconContainerReconciler) newConfigMap(ctx context.Context, falconContainer *v1alpha1.FalconContainer) (*corev1.ConfigMap, error) {
	data := make(map[string]string)
	data["CP_NAMESPACE"] = r.Namespace()
	data["FALCON_INJECTOR_LISTEN_PORT"] = strconv.Itoa(int(*falconContainer.Spec.Injector.ListenPort))

	imageUri, err := r.imageUri(ctx, falconContainer)
	if err != nil {
		r.Log.Error(err, "unable to determine falcon-container image URI")
	} else {
		data["FALCON_IMAGE"] = imageUri
	}

	data["FALCON_IMAGE_PULL_POLICY"] = string(falconContainer.Spec.Injector.ImagePullPolicy)

	data["FALCON_IMAGE_PULL_SECRET"] = falconContainer.Spec.Injector.ImagePullSecretName

	if falconContainer.Spec.Injector.DisableDefaultPodInjection {
		data["INJECTION_DEFAULT_DISABLED"] = "T"
	}
	cid, err := falcon_api.FalconCID(ctx, falconContainer.Spec.FalconAPI.CID, falconContainer.Spec.FalconAPI.ApiConfig())
	if err != nil {
		return &corev1.ConfigMap{}, fmt.Errorf("unable to determine falcon customer ID (CID): %v", err)
	}
	data["FALCONCTL_OPT_CID"] = cid

	if falconContainer.Spec.Injector.LogVolume != nil {

		vol, err := common.EncodeBase64Interface(*falconContainer.Spec.Injector.LogVolume)
		if err != nil {
			r.Log.Error(err, "unable to base64 encode log volume")
		}
		data["FALCON_LOG_VOLUME"] = vol
	}

	if falconContainer.Spec.Injector.SensorResources != nil {

		resources, err := common.EncodeBase64Interface(*falconContainer.Spec.Injector.SensorResources)
		if err != nil {
			r.Log.Error(err, "unable to base64 encode falcon resources")
		}
		data["FALCON_RESOURCES"] = resources
	}

	if falconContainer.Spec.Injector.FalconctlOpts != "" {
		data["FALCONCTL_OPTS"] = falconContainer.Spec.Injector.FalconctlOpts
	}

	if falconContainer.Spec.Injector.AdditionalEnvironmentVariables != nil {
		for k, v := range *falconContainer.Spec.Injector.AdditionalEnvironmentVariables {
			data[strings.ToUpper(k)] = v
		}
	}

	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ConfigMap",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      injectorConfigMapName,
			Namespace: r.Namespace(),
			Labels:    FcLabels,
		},
		Data: data,
	}, nil
}
