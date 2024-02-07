package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// FalconImageSpec defines the desired state of FalconImage
type FalconImageSpec struct {
	// Namespace where the Falcon Image Controller should be installed.
	// For best security practices, this should be a dedicated namespace that is not used for any other purpose.
	// It also should not be the same namespace where the Falcon Operator or the Falcon Sensor is installed.
	// +kubebuilder:default:=falcon-imageanalyzer
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=1,xDescriptors={"urn:alm:descriptor:io.kubernetes:Namespace"}
	InstallNamespace string `json:"installNamespace,omitempty"`

	// Additional configuration for Falcon Image Controller deployment.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Image Controller Configuration",order=5
	ImageConfig FalconImageConfigSpec `json:"ImageConfig,omitempty"`
}

type FalconImageConfigSpec struct {
	// Define annotations that will be passed down to admision controller service account. This is useful for passing along AWS IAM Role or GCP Workload Identity.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Service Account Configuration",order=7
	ServiceAccount FalconImageServiceAccount `json:"serviceAccount,omitempty"`

	// +kubebuilder:default:=Always
	// +kubebuilder:validation:Enum=Always;IfNotPresent;Never
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Container Image Pull Policy",order=4
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// ImagePullSecrets is an optional list of references to secrets to use for pulling image from the image location.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=1,displayName="Falcon Image Controller Image Pull Secrets",xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret"}
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
}

type FalconImageServiceAccount struct {
	// Define annotations that will be passed down to the Service Account. This is useful for passing along AWS IAM Role or GCP Workload Identity.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Service Account Annotations",order=1
	Annotations map[string]string `json:"annotations,omitempty"`
}

// FalconImageStatus defines the observed state of FalconImage
type FalconImageStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Version of the CrowdStrike Falcon Sensor
	// +operator-sdk:csv:customresourcedefinitions:type=status,displayName="Falcon Sensor Version",xDescriptors={"urn:alm:descriptor:text"}
	Sensor *string `json:"sensor,omitempty"`

	// Version of the CrowdStrike Falcon Operator
	// +operator-sdk:csv:customresourcedefinitions:type=status,displayName="Falcon Operator Version",xDescriptors={"urn:alm:descriptor:text"}
	Version string `json:"version,omitempty"`

	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=status,displayName="Falcon Image Conditions",xDescriptors={"urn:alm:descriptor:io.kubernetes.conditions"}
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster
//+kubebuilder:printcolumn:name="Operator Version",type="string",JSONPath=".status.version",description="Version of the Operator"
//+kubebuilder:printcolumn:name="Falcon Sensor",type="string",JSONPath=".status.sensor",description="Version of the Falcon Image Analyzer"

// FalconImage is the Schema for the falconimages API
type FalconImage struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FalconImageSpec `json:"spec,omitempty"`
	Status FalconCRStatus  `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// FalconImageList contains a list of FalconImage
type FalconImageList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FalconImage `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FalconImage{}, &FalconImageList{})
}
