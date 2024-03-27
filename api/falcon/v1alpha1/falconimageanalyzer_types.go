package v1alpha1

import (
	arv1 "k8s.io/api/admissionregistration/v1"
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
	// +kubebuilder:default:=falcon-image-analyzer
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=1,xDescriptors={"urn:alm:descriptor:io.kubernetes:Namespace"}
	InstallNamespace string `json:"installNamespace,omitempty"`

	// CrowdStrike Falcon sensor configuration
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Sensor Configuration",order=3
	Falcon FalconSensor `json:"falcon,omitempty"`

	// FalconAPI configures connection from your local Falcon operator to CrowdStrike Falcon platform.
	//
	// When configured, it will pull the sensor from registry.crowdstrike.com and deploy the appropriate sensor to the cluster.
	//
	// If using the API is not desired, the sensor can be manually configured by setting the Image and Version fields.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Platform API Configuration",order=2
	FalconAPI *FalconAPI `json:"falcon_api,omitempty"`

	// ResourceQuota configures the ResourceQuota for the Falcon Image Analyzer. This is useful for limiting the number of pods that can be created in the namespace.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Image Analyzer Resource Quota",order=4
	ResQuota FalconImageAnalyzerRQSpec `json:"resourcequota,omitempty"`

	// Registry configures container image registry to which the Image Analyzer image will be pushed.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Image Analyzer Registry Configuration",order=6
	Registry RegistrySpec `json:"registry,omitempty"`

	// Additional configuration for Falcon Image Analyzer deployment.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Image Analyzer Configuration",order=5
	ImageAnalyzerConfig FalconImageAnalyzerConfigSpec `json:"imageAnalyzerConfig,omitempty"`

	// Location of the Falcon Sensor image. Use only in cases when you mirror the original image to your repository/name:tag, and CrowdStrike OAuth2 API is not used.
	// +kubebuilder:validation:Pattern="^.*:.*$"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Image Analyzer Image URI",order=7
	Image string `json:"image,omitempty"`

	// Falcon Image Analyzer Version. The latest version will be selected when version specifier is missing. Example: 6.31, 6.31.0, 6.31.0-1409, etc.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Image Analyzer Version",order=8
	Version *string `json:"version,omitempty"`
}

type FalconImageAnalyzerRQSpec struct {
	// Limits the number of Image Analyzer pods that can be created in the namespace.
	// +kubebuilder:default:="2"
	// +kubebuilder:validation:String
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Resource Quota Pod Limit",order=1,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:podCount"}
	PodLimit string `json:"pods,omitempty"`
}

type FalconImageAnalyzerConfigSpec struct {
	// Define annotations that will be passed down to admision controller service account. This is useful for passing along AWS IAM Role or GCP Workload Identity.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Service Account Configuration",order=7
	ServiceAccount FalconImageAnalyzerServiceAccount `json:"serviceAccount,omitempty"`

	// Port on which the Falcon Image Analyzer service will listen for requests from the cluster.
	// +kubebuilder:default:=443
	// +kubebuilder:validation:XIntOrString
	// +kubebuilder:validation:Minimum:=0
	// +kubebuilder:validation:Maximum:=65535
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Image Analyzer Service Port",order=3,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number"}
	Port *int32 `json:"servicePort,omitempty"`

	// Port on which the Falcon Image Analyzer container will listen for requests.
	// +kubebuilder:default:=4443
	// +kubebuilder:validation:XIntOrString
	// +kubebuilder:validation:Minimum:=0
	// +kubebuilder:validation:Maximum:=65535
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Image Analyzer Container Port",order=4,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number"}
	ContainerPort *int32 `json:"containerPort,omitempty"`

	// Configure the failure policy for the Falcon Image Analyzer.
	// +kubebuilder:default:=Ignore
	// +kubebuilder:validation:Enum=Ignore;Fail
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Image Analyzer Failure Policy",order=6
	FailurePolicy arv1.FailurePolicyType `json:"failurePolicy,omitempty"`

	// Ignore Image Analyzer for a specific set of namespaces.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Ignore Namespace List",order=12
	DisabledNamespaces FalconImageAnalyzerNamespace `json:"disabledNamespaces,omitempty"`

	// Number of replicas for the Falcon Image Analyzer deployment.
	// +kubebuilder:default:=2
	// +kubebuilder:validation:XIntOrString
	// +kubebuilder:validation:Minimum:=0
	// +kubebuilder:validation:Maximum:=65535
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Image Analyzer Replica Count",order=5,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number"}
	Replicas *int32 `json:"replicas,omitempty"`

	// +kubebuilder:default:=Always
	// +kubebuilder:validation:Enum=Always;IfNotPresent;Never
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Image Analyzer Image Pull Policy",order=2,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:imagePullPolicy"}
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// ImagePullSecrets is an optional list of references to secrets to use for pulling image from the image location.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=1,displayName="Falcon Image Analyzer Image Pull Secrets",xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret"}
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Image Analyzer Client Resources",order=9,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:resourceRequirements"}
	//+kubebuilder:default:={"limits":{"cpu":"750m","memory":"256Mi"},"requests":{"cpu":"500m","memory":"256Mi"}}
	ResourcesClient *corev1.ResourceRequirements `json:"resourcesClient,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Image Analyzer Resources",order=10,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:resourceRequirements"}
	//+kubebuilder:default:={"limits":{"cpu":"300m","memory":"512Mi"},"requests":{"cpu":"300m","memory":"512Mi"}}
	ResourcesAC *corev1.ResourceRequirements `json:"resources,omitempty"`

	// Type of Deployment update. Can be "RollingUpdate" or "OnDelete". Default is RollingUpdate.
	// +kubebuilder:default:={"rollingUpdate":{"maxUnavailable":0,"maxSurge":1}}
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Deployment Update Strategy",order=11
	DepUpdateStrategy FalconImageAnalyzerUpdateStrategy `json:"updateStrategy,omitempty"`
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

type FalconImageAnalyzerNamespace struct {
	// Configure a list of namespaces to ignore Image Analyzer.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Ignore Namespace List",order=1
	Namespaces []string `json:"namespaces,omitempty"`

	// For OpenShift clusters, ignore openshift-specific namespaces for Image Analyzer.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Ignore OpenShift Namespaces",order=2,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	IgnoreOpenShiftNamespaces bool `json:"ignoreOpenShiftNamespaces,omitempty"`
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
