package v1alpha1

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// FalconNodeSensorSpec defines the desired state of FalconNodeSensor
// +k8s:openapi-gen=true
type FalconNodeSensorSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Namespace where the Falcon Sensor should be installed.
	// For best security practices, this should be a dedicated namespace that is not used for any other purpose.
	// It also should not be the same namespace where the Falcon Operator, or other Falcon resources are deployed.
	// +kubebuilder:default:=falcon-system
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=1,xDescriptors={"urn:alm:descriptor:io.kubernetes:Namespace"}
	InstallNamespace string `json:"installNamespace,omitempty"`

	// Various configuration for DaemonSet Deployment
	// +kubebuilder:default:={}
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="DaemonSet Configuration",order=3
	Node FalconNodeSensorConfig `json:"node,omitempty"`

	// +kubebuilder:default:={}
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
	// +optional
	// +kubebuilder:default:={{key: "node-role.kubernetes.io/master", operator: "Exists", effect: "NoSchedule"}, {key: "node-role.kubernetes.io/control-plane", operator: "Exists", effect: "NoSchedule"}, {key: "node-role.kubernetes.io/infra", operator: "Exists", effect: "NoSchedule"}}
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=4
	Tolerations *[]corev1.Toleration `json:"tolerations"`

	// Specifies node affinity for scheduling the DaemonSet. Defaults to allowing scheduling on all nodes.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=5
	NodeAffinity corev1.NodeAffinity `json:"nodeAffinity,omitempty"`

	// +kubebuilder:default=Always
	// +kubebuilder:validation:Enum=Always;IfNotPresent;Never
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=3
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// Location of the Falcon Sensor image. Use only in cases when you mirror the original image to your repository/name:tag
	// +kubebuilder:validation:Pattern="^.*:.*$"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=2
	Image string `json:"image,omitempty"`

	// ImagePullSecrets is an optional list of references to secrets in the falcon-system namespace to use for pulling image from image_override location.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,order=1
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// Type of DaemonSet update. Can be "RollingUpdate" or "OnDelete". Default is RollingUpdate.
	// +kubebuilder:default={}
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

	// Configure resource requests and limits for the DaemonSet Sensor. Only applies when using the eBPF backend.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon eBPF Sensor Resources",order=9
	SensorResources Resources `json:"resources,omitempty"`

	// Sets the backend to be used by the DaemonSet Sensor.
	// +kubebuilder:default=bpf
	// +kubebuilder:validation:Enum=kernel;bpf
	// +operator-sdk-csv:customresourcedefinitions:type=spec,order=10
	Backend string `json:"backend,omitempty"`

	// Enables the use of GKE Autopilot.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="GKE Autopilot Settings",order=11
	GKE AutoPilot `json:"gke,omitempty"`

	// Enable priority class for the DaemonSet. This is useful for GKE Autopilot clusters, but can be set for any cluster.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Priority Class",order=12
	PriorityClass PriorityClassConfig `json:"priorityClass,omitempty"`

	// Version of the sensor to be installed. The latest version will be selected when this version specifier is missing.
	Version *string `json:"version,omitempty"`

	// Advanced configures various options that go against industry practices or are otherwise not recommended for use.
	// Adjusting these settings may result in incorrect or undesirable behavior. Proceed at your own risk.
	// For more information, please see https://github.com/CrowdStrike/falcon-operator/blob/main/docs/ADVANCED.md.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="DaemonSet Advanced Settings"
	Advanced FalconAdvanced `json:"advanced,omitempty"`
}

type PriorityClassConfig struct {
	// Enables the operator to deploy a PriorityClass instead of rolling your own. Default is false.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Deploy Priority Class to cluster",order=2
	Deploy *bool `json:"deploy,omitempty"`

	// Name of the priority class to use for the DaemonSet.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Name of the Priority Class to use",order=2
	Name string `json:"name,omitempty"`

	// Value of the priority class to use for the DaemonSet. Requires the Deploy field to be set to true.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Priority Class Value",order=3
	Value *int32 `json:"value,omitempty"`
}

type Resources struct {
	// Sets the resource limits for the DaemonSet Sensor. Only applies when using the eBPF backend.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Limits ResourceList `json:"limits,omitempty"`

	// Sets the resource requests for the DaemonSet Sensor. Only applies when using the eBPF backend.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Requests ResourceList `json:"requests,omitempty"`
}

type ResourceList struct {
	// Minimum allowed is 250m.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Pattern="^(([0-9]{4,}|[2-9][5-9][0-9])m$)|[0-9]+$"
	CPU string `json:"cpu,omitempty"`

	// Minimum allowed is 500Mi.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	// +kubebuilder:validation:Pattern="^(([5-9][0-9]{2}[Mi]+)|([0-9.]+[iEGTP]+))|(([5-9][0-9]{8})|([0-9]{10,}))$"
	Memory string `json:"memory,omitempty"`

	// +operator-sdk:csv:customresourcedefinitions:type=spec
	EphemeralStorage string `json:"ephemeral-storage,omitempty"`
}

type AutoPilot struct {
	// Enables the use of GKE Autopilot.
	// +operator-sdk:csv:customresourcedefinitions:type=spec
	Enabled *bool `json:"autopilot,omitempty"`
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
//+kubebuilder:printcolumn:name="Operator Version",type="string",JSONPath=".status.version",description="Version of the Operator"
//+kubebuilder:printcolumn:name="Falcon Sensor",type="string",JSONPath=".status.sensor",description="Version of the Falcon Sensor"

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

func (node FalconNodeSensor) GetTolerations() *[]corev1.Toleration {
	if node.Spec.Node.Tolerations == nil {
		return &[]corev1.Toleration{}
	}
	return node.Spec.Node.Tolerations
}
