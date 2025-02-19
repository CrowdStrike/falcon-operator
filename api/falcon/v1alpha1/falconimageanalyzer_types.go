package v1alpha1

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// FalconImageAnalyzerSpec defines the desired state of FalconImageAnalyzer
type FalconImageAnalyzerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Namespace where the Falcon Image Analyzer should be installed.
	// For best security practices, this should be a dedicated namespace that is not used for any other purpose.
	// It also should not be the same namespace where the Falcon Operator or the Falcon Sensor is installed.
	// +kubebuilder:default:=falcon-iar
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=1,xDescriptors={"urn:alm:descriptor:io.kubernetes:Namespace"}
	InstallNamespace string `json:"installNamespace,omitempty"`

	// FalconAPI configures connection from your local Falcon operator to CrowdStrike Falcon platform.
	//
	// When configured, it will pull the sensor from registry.crowdstrike.com and deploy the appropriate sensor to the cluster.
	//
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Platform API Configuration",order=2
	FalconAPI *FalconAPI `json:"falcon_api,omitempty"`

	// Registry configures container image registry to which the Image Analyzer image will be pushed.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Image Analyzer Registry Configuration",order=6
	Registry RegistrySpec `json:"registry,omitempty"`

	// Additional configuration for Falcon Image Analyzer deployment.
	// +kubebuilder:default:={}
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Image Analyzer Configuration",order=5
	ImageAnalyzerConfig FalconImageAnalyzerConfigSpec `json:"imageAnalyzerConfig,omitempty"`

	// Location of the Image Analyzer image. Use only in cases when you mirror the original image to your repository/name:tag
	// +kubebuilder:validation:Pattern="^.*:.*$"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Image Analyzer Image URI",order=7
	Image string `json:"image,omitempty"`

	// Falcon Image Analyzer Version. The latest version will be selected when version specifier is missing. Example: 6.31, 6.31.0, 6.31.0-1409, etc.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Image Analyzer Version",order=8
	Version *string `json:"version,omitempty"`
}

type FalconImageAnalyzerConfigSpec struct {
	// Define annotations that will be passed down to Image Analyzer service account. This is useful for passing along AWS IAM Role or GCP Workload Identity.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Service Account Configuration",order=1
	ServiceAccount FalconImageAnalyzerServiceAccount `json:"serviceAccount,omitempty"`

	// +kubebuilder:default:=Always
	// +kubebuilder:validation:Enum=Always;IfNotPresent;Never
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Image Analyzer Image Pull Policy",order=2,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:imagePullPolicy"}
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// ImagePullSecrets is an optional list of references to secrets to use for pulling image from the image location.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=3,displayName="Falcon Image Analyzer Image Pull Secrets",xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret"}
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Image Analyzer Resources",order=4,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:resourceRequirements"}
	Resources *corev1.ResourceRequirements `json:"resources,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Azure Config file path",order=5
	AzureConfigPath string `json:"azureConfigPath,omitempty"`

	// Enable priority class for the Falcon Image Analyzer deployment.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Priority Class",order=6
	PriorityClass FalconImageAnalyzerPriorityClass `json:"priorityClass,omitempty"`

	// Type of Deployment update. Can be "RollingUpdate" or "OnDelete". Default is RollingUpdate.
	// +kubebuilder:default:={"rollingUpdate":{"maxUnavailable":0,"maxSurge":1}}
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Deployment Update Strategy",order=7
	DepUpdateStrategy FalconImageAnalyzerUpdateStrategy `json:"updateStrategy,omitempty"`

	// Set the falcon image analyzer volume size limit.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Image Analyzer Volume Size Limit",order=8
	// +kubebuilder:default:="20Gi"
	VolumeSizeLimit string `json:"sizeLimit,omitempty"`

	// Set the falcon image analyzer volume mount path.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Image Analyzer Volume Mount Path",order=9
	// +kubebuilder:default:="/tmp"
	VolumeMountPath string `json:"mountPath,omitempty"`

	// Name of the Kubernetes Cluster.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Image Analyzer Cluster Name",order=10
	ClusterName string `json:"clusterName,omitempty"`

	// Exclusions for the Falcon Image Analyzer.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Image Analyzer Exclusions",order=11
	Exclusions Exclusions `json:"exclusions,omitempty"`

	// RegistryConfig for the Falcon Image Analyzer.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Image Analyzer Registry Configuration Options",order=12
	RegistryConfig RegistryConfig `json:"registryConfig,omitempty"`

	// Enable debugging for the Falcon Image Analyzer.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Image Analyzer Enable Debugging",order=13
	// +kubebuilder:default:=false
	EnableDebug bool `json:"debug,omitempty"`
}

type FalconImageAnalyzerPriorityClass struct {
	// Name of the priority class to use.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Name of the Priority Class to use",order=1
	Name string `json:"name,omitempty"`
}

type FalconImageAnalyzerServiceAccount struct {
	// Define annotations that will be passed down to the Service Account. This is useful for passing along AWS IAM Role or GCP Workload Identity.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Service Account Annotations",order=1
	Annotations map[string]string `json:"annotations,omitempty"`
}

type FalconImageAnalyzerUpdateStrategy struct {
	// RollingUpdate is used to specify the strategy used to roll out a deployment
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Admisison Controller deployment update configuration",order=1,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:updateStrategy"}
	RollingUpdate appsv1.RollingUpdateDeployment `json:"rollingUpdate,omitempty"`
}

type Exclusions struct {
	// Configure a list of registries for the Falcon Image Analyzer to ignore.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Exclusions List",order=1
	Registries []string `json:"registries,omitempty"`

	// Configure a list of namespaces for Image Analyzer to ignore.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Ignore Namespace List",order=2
	Namespaces []string `json:"namespaces,omitempty"`
}

type RegistryConfig struct {
	// If neceeary, configure the registry credentials for the Falcon Image Analyzer.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Registry Credentials",order=1
	Credentials []RegistryCreds `json:"credentials,omitempty"`
}

type RegistryCreds struct {
	// Namespace where the registry container secret is located.
	Namespace string `json:"namespace,omitempty"`
	// Name of the registry container secret.
	SecretName string `json:"secretName,omitempty"`
}

// FalconImageAnalyzerStatus defines the observed state of FalconImageAnalyzer
type FalconImageAnalyzerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Version of the CrowdStrike Falcon Sensor
	// +operator-sdk:csv:customresourcedefinitions:type=status,displayName="Falcon Sensor Version",xDescriptors={"urn:alm:descriptor:text"}
	Sensor *string `json:"sensor,omitempty"`

	// Version of the CrowdStrike Falcon Operator
	// +operator-sdk:csv:customresourcedefinitions:type=status,displayName="Falcon Operator Version",xDescriptors={"urn:alm:descriptor:text"}
	Version string `json:"version,omitempty"`

	// +optional
	// +operator-sdk:csv:customresourcedefinitions:type=status,displayName="Falcon Image Analyzer Conditions",xDescriptors={"urn:alm:descriptor:io.kubernetes.conditions"}
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster
//+kubebuilder:printcolumn:name="Operator Version",type="string",JSONPath=".status.version",description="Version of the Operator"
//+kubebuilder:printcolumn:name="Falcon Sensor",type="string",JSONPath=".status.sensor",description="Version of the Falcon Image Analyzer"

// FalconImageAnalyzer is the Schema for the falconImageAnalyzers API
type FalconImageAnalyzer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FalconImageAnalyzerSpec `json:"spec,omitempty"`
	Status FalconCRStatus          `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// FalconImageAnalyzerList contains a list of FalconImageAnalyzer
type FalconImageAnalyzerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FalconImageAnalyzer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FalconImageAnalyzer{}, &FalconImageAnalyzerList{})
}
