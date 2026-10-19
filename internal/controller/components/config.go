package components

import (
	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// BaseConfig holds fields common to all sensor sub-reconcilers.
type BaseConfig struct {
	Request          ctrl.Request
	Owner            client.Object
	Status           *falconv1alpha1.FalconCRStatus
	InstallNamespace string
	Image            string
	ImagePullPolicy  corev1.PullPolicy
	ImagePullSecrets []corev1.LocalObjectReference
	NamePrefix       string
	Cid              string
	Falcon           falconv1alpha1.FalconSensor
	OpenShift        bool
}
