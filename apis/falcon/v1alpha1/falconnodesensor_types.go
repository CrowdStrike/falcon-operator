package v1alpha1

import (
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

	Node   FalconNodeSensorConfig `json:"node"`
	Falcon FalconSensor           `json:"falcon"`
	// FalconAPI configures connection from your local Falcon operator to CrowdStrike Falcon platform.
	FalconAPI *FalconAPI `json:"falcon_api,omitempty"`
}

// CrowdStrike Falcon Sensor configuration settings.
// +k8s:openapi-gen=true
type FalconSensor struct {
	// Falcon Customer ID (CID)
	// +kubebuilder:validation:Pattern="^[0-9a-fA-F]{32}-[0-9a-fA-F]{2}$"
	CID *string `json:"cid,omitempty"`
	// Enable the App Proxy Port (APP). Uncommon in container-based deployments.
	APD *bool `json:"apd,omitempty"`
	// App Proxy Hostname (APH). Uncommon in container-based deployments.
	APH string `json:"aph,omitempty"`
	// App Proxy Port (APP). Uncommon in container-based deployments.
	APP *int `json:"app,omitempty"`
	// Utilize default or metered billing.
	// +kubebuilder:validation:Enum=default;metered
	Billing string `json:"billing,omitempty"`
	// Provisioning token.
	// +kubebuilder:validation:Pattern="^[0-9a-fA-F]{8}$"
	PToken string `json:"provisioning_token,omitempty"`
	// List of tags for sensor grouping. Allowed characters: all alphanumerics, '/', '-', and '_'.
	Tags []string `json:"tags,omitempty"`
	// Set trace level. Options are [none|err|warn|info|debug].
	// +kubebuilder:validation:Enum=none;err;warn;info;debug
	Trace string `json:"trace,omitempty"`
}

// FalconNodeSensorConfig defines aspects about how the daemonset works.
// +k8s:openapi-gen=true
type FalconNodeSensorConfig struct {
	// Specifies tolerations for custom taints. Defaults to allowing scheduling on all nodes.
	// +kubebuilder:default={{key: "node-role.kubernetes.io/master", operator: "Exists", effect: "NoSchedule"}, {key: "node-role.kubernetes.io/control-plane", operator: "Exists", effect: "NoSchedule"}}
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
	// +kubebuilder:default=Always
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
	// Location of the Falcon Sensor image. Use only in cases when you mirror the original image to your repository/name:tag
	ImageOverride string `json:"image_override,omitempty"`
	// ImagePullSecrets is an optional list of references to secrets in the falcon-system namespace to use for pulling image from image_override location.
	ImagePullSecrets []corev1.LocalObjectReference `json:"imagePullSecrets,omitempty"`
	// Kills pod after a specificed amount of time (in seconds). Default is 30 seconds.
	// +kubebuilder:default=30
	TerminationGracePeriod int64 `json:"terminationGracePeriod,omitempty"`
	// Disables the cleanup of the sensor through DaemonSet on the nodes.
	// Disabling might have unintended consequences for certain operations such as sensor downgrading.
	// +kubebuilder:default=false
	NodeCleanup *bool `json:"disableCleanup,omitempty"`
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
