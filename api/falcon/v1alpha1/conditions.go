package v1alpha1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

const (
	// Following strings are condition types

	ConditionUnknown         string = "Unknown"
	ConditionSuccess         string = "Success"
	ConditionFailed          string = "Failed"
	ConditionPending         string = "Pending"
	ConditionImageReady      string = "ImageReady"
	ConditionConfigMapReady  string = "ConfigMapReady"
	ConditionDaemonSetReady  string = "DaemonSetReady"
	ConditionDeploymentReady string = "DeploymentReady"
	ConditionServiceReady    string = "ServiceReady"
	ConditionRouteReady      string = "RouteReady"
	ConditionSecretReady     string = "SecretReady"
	ConditionWebhookReady    string = "WebhookReady"

	// Following strings are condition reasons

	ReasonReqNotMet        string = "RequirementsNotMet"
	ReasonReqMet           string = "RequirementsMet"
	ReasonInstallSucceeded string = "InstallSucceeded"
	ReasonInstallFailed    string = "InstallFailed"
	ReasonSucceeded        string = "Succeeded"
	ReasonUpdateSucceeded  string = "UpdateSucceeded"
	ReasonUpdateFailed     string = "UpdateFailed"
	ReasonDeleteSucceeded  string = "DeleteSucceeded"
	ReasonDeleteFailed     string = "DeleteFailed"
	ReasonFailed           string = "Failed"
	ReasonDiscovered       string = "Discovered"
)

// FalconAdmissionStatus defines the observed state of FalconAdmission
type FalconCRStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Version of the CrowdStrike Falcon Sensor
	Sensor *string `json:"sensor,omitempty"`

	// Version of the CrowdStrike Falcon Operator
	Version string `json:"version,omitempty"`

	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}
