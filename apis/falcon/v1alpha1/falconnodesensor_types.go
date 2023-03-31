package v1alpha1

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

const (
	// Following strings are condition types

	ConditionSuccess        string = "Success"
	ConditionFailed         string = "Failed"
	ConditionPending        string = "Pending"
	ConditionConfigMapReady string = "ConfigMapReady"
	ConditionDaemonSetReady string = "DaemonSetReady"

	// Following strings are condition reasons

	ReasonReqNotMet        string = "RequirementsNotMet"
	ReasonInstallSucceeded string = "InstallSucceeded"
	ReasonInstallFailed    string = "InstallFailed"
	ReasonSucceeded        string = "Succeeded"
	ReasonUpdateSucceeded  string = "UpdateSucceeded"
	ReasonUpdateFailed     string = "UpdateFailed"
	ReasonFailed           string = "Failed"
)

// FalconNodeSensorSpec defines the desired state of FalconNodeSensor
// +k8s:openapi-gen=true
type FalconNodeSensorSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Various configuration for DaemonSet Deployment
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="DaemonSet Configuration",order=3
	Node FalconNodeSensorConfig `json:"node,omitempty"`
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Sensor Configuration",order=2
	Falcon FalconSensor `json:"falcon,omitempty"`
	// FalconAPI configures connection from your local Falcon operator to CrowdStrike Falcon platform.
	//
	// When configured, it will pull the sensor from registry.crowdstrike.com and deploy the appropriate sensor to the cluster.
	//
	// If using the API is not desired, the sensor can be manually configured.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Platform API Configuration",order=1
	FalconAPI *FalconAPI `json:"falcon_api,omitempty"`
}

// FalconNodeSensorConfig defines aspects about how the daemonset works.
// +k8s:openapi-gen=true
type FalconNodeSensorConfig struct {
	// Specifies tolerations for custom taints. Defaults to allowing scheduling on all nodes.
	// +kubebuilder:default:={{key: "node-role.kubernetes.io/master", operator: "Exists", effect: "NoSchedule"}, {key: "node-role.kubernetes.io/control-plane", operator: "Exists", effect: "NoSchedule"}}
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=4
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
	// Specifies node affinity for scheduling the DaemonSet. Defaults to allowing scheduling on all nodes.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=5
	NodeAffinity corev1.NodeAffinity `json:"nodeAffinity,omitempty"`
	// +kubebuilder:default=Always
	// +kubebuilder:validation:Enum=Always;IfNotPresent;Never
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=3
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
	// Location of the Falcon Sensor image. Use only in cases when you mirror the original image to your repository/name:tag
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=2
	ImageOverride string `json:"image_override,omitempty"`
	// ImagePullSecrets is an optional list of references to secrets in the falcon-system namespace to use for pulling image from image_override location.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=1
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	// Type of DaemonSet update. Can be "RollingUpdate" or "OnDelete". Default is RollingUpdate.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="DaemonSet Update Strategy",order=6
	DSUpdateStrategy FalconNodeUpdateStrategy `json:"updateStrategy,omitempty"`
	// Kills pod after a specificed amount of time (in seconds). Default is 30 seconds.
	// +kubebuilder:default:=30
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=7
	TerminationGracePeriod int64 `json:"terminationGracePeriod,omitempty"`
	// Add metadata to the DaemonSet Service Account for IAM roles.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	ServiceAccount FalconNodeServiceAccount `json:"serviceAccount,omitempty"`
	// Disables the cleanup of the sensor through DaemonSet on the nodes.
	// Disabling might have unintended consequences for certain operations such as sensor downgrading.
	// +kubebuilder:default=false
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=8
	NodeCleanup *bool `json:"disableCleanup,omitempty"`
	// Sets the backend to be used by the DaemonSet Sensor.
	// +kubebuilder:default=kernel
	// +kubebuilder:validation:Enum=kernel;bpf
	// +operator-sdk-csv:customresourcedefinitions:type=spec,order=9
	Backend string `json:"backend,omitempty"`

	// Version of the sensor to be installed. The latest version will be selected when this version specifier is missing.
	Version *string `json:"version,omitempty"`
}

type FalconNodeUpdateStrategy struct {
	// +kubebuilder:default=RollingUpdate
	// +kubebuilder:validation:Enum=RollingUpdate;OnDelete
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Type          appsv1.DaemonSetUpdateStrategyType `json:"type,omitempty"`
	RollingUpdate appsv1.RollingUpdateDaemonSet      `json:"rollingUpdate,omitempty"`
}

type FalconNodeServiceAccount struct {
	// Define annotations that will be passed down to the Service Account. This is useful for passing along AWS IAM Role or GCP Workload Identity.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Annotations map[string]string `json:"annotations,omitempty"`
}

// FalconNodeSensorStatus defines the observed state of FalconNodeSensor
// +k8s:openapi-gen=true
type FalconNodeSensorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// Phase or the status of the deployment

	// Version of the CrowdStrike Falcon Sensor
	Sensor *string `json:"sensor,omitempty"`

	// Version of the CrowdStrike Falcon Operator
	Version string `json:"version,omitempty"`

	// Conditions represent the latest available observations of an object's state
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster
//+kubebuilder:printcolumn:name="Falcon Sensor",type="string",JSONPath=".status.sensor"
//+kubebuilder:printcolumn:name="Operator Version",type="string",JSONPath=".status.version"

// FalconNodeSensor is the Schema for the falconnodesensors API
// +k8s:openapi-gen=true
type FalconNodeSensor struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              FalconNodeSensorSpec   `json:"spec,omitempty"`
	Status            FalconNodeSensorStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// FalconNodeSensorList contains a list of FalconNodeSensor
type FalconNodeSensorList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FalconNodeSensor `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FalconNodeSensor{}, &FalconNodeSensorList{})
}
