package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

type FalconNodeSensorStatusPhase string

const (
	NodePhaseInitializing FalconNodeSensorStatusPhase = "Initializing"
	NodePhaseActive       FalconNodeSensorStatusPhase = "Active"
	NodePhasePending      FalconNodeSensorStatusPhase = "Pending"
	NodePhaseError        FalconNodeSensorStatusPhase = "Error"
)

type FalconNodeSensorCondition string

const (
	NodeConditionSucceeded FalconNodeSensorCondition = "Succeeded"
	NodeConditionFailed    FalconNodeSensorCondition = "Failed"
	NodeConditionErrored   FalconNodeSensorCondition = "Errored"
)

// FalconNodeSensorSpec defines the desired state of FalconNodeSensor
// +k8s:openapi-gen=true
type FalconNodeSensorSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Node   FalconNodeSensorConfig `json:"node"`
	Falcon FalconSensor           `json:"falcon"`
}

// CrowdStrike Falcon Sensor configuration settings.
// +k8s:openapi-gen=true
type FalconSensor struct {
	// Falcon Customer ID (CID)
	// +kubebuilder:validation:Pattern="^[0-9a-fA-F]{32}-[0-9a-fA-F]{2}$"
	CID string `json:"cid"`
	// Enable the App Proxy Port (APP). Uncommon in container-based deployments.
	APD bool `json:"apd,omitempty"`
	// App Proxy Hostname (APH). Uncommon in container-based deployments.
	APH string `json:"aph,omitempty"`
	// App Proxy Port (APP). Uncommon in container-based deployments.
	APP string `json:"app,omitempty"`
	// Utilize default or metered billing.
	// +kubebuilder:validation:Enum=default;metered
	Billing bool `json:"billing,omitempty"`
	// Options to pass to the "--feature" flag. Options are [none,[enableLog[,disableLogBuffer[,disableOsfm[,emulateUpdate]]]]].
	Feature string `json:"feature,omitempty"`
	// Enable message log for logging to disk.
	MessageLog bool `json:"message_log,omitempty"`
	// Provisioning token.
	// +kubebuilder:validation:Pattern="^[0-9a-fA-F]{8}$"
	PToken string `json:"provisioning_token,omitempty"`
	// List of tags for sensor grouping. Allowed characters: all alphanumerics, '/', '-', and '_'.
	// +kubebuilder:validation:Pattern=`^[a-zA-Z0-9\/\-_\,]+$`
	Tags string `json:"tags,omitempty"`
	// Set trace level. Options are [none|err|warn|info|debug].
	// +kubebuilder:validation:Enum=none;err;warn;info;debug
	Trace string `json:"trace,omitempty"`
}

// FalconNodeSensorConfig defines aspects about how the daemonset works.
// +k8s:openapi-gen=true
type FalconNodeSensorConfig struct {
	// Specifies tolerations for custom taints. Defaults to allowing scheduling on all nodes.
	// +kubebuilder:default={{operator: "Exists", effect: "NoSchedule"}, {operator: "Exists", effect: "NoExecute"}}
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
	// +kubebuilder:default=Always
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
	// Name of the Falcon Sensor container to pull. Format should be repository/namespace/name:tag
	// +kubebuilder:default="falcon-node-sensor:latest"
	Image string `json:"image,omitempty"`
	// Kills pod after a specificed amount of time (in seconds). Default is 30 seconds.
	// +kubebuilder:default=30
	TerminationGracePeriod int64 `json:"terminationGracePeriod,omitempty"`
}

// FalconNodeSensorStatus defines the observed state of FalconNodeSensor
// +k8s:openapi-gen=true
type FalconNodeSensorStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

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
