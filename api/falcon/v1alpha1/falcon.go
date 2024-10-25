package v1alpha1

const (
	TraceDefault = "none"
)

// CrowdStrike Falcon Sensor configuration settings.
// +k8s:openapi-gen=true
type FalconSensor struct {
	// Falcon Customer ID (CID)
	// +kubebuilder:validation:Pattern:="^[0-9a-fA-F]{32}-[0-9a-fA-F]{2}$"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Customer ID (CID)",order=1
	CID *string `json:"cid,omitempty"`

	// Disable the Falcon Sensor's use of a proxy.
	// +kubebuilder:default:=false
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Disable Falcon Proxy",order=3,xDescriptors="urn:alm:descriptor:com.tectonic.ui:booleanSwitch"
	APD *bool `json:"apd,omitempty"`

	// The application proxy host to use for Falcon sensor proxy configuration.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Proxy Host",order=4
	APH string `json:"aph,omitempty"`

	// The application proxy port to use for Falcon sensor proxy configuration.
	// +kubebuilder:validation:Minimum:=0
	// +kubebuilder:validation:Maximum:=65535
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Proxy Port",order=5,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:number"}
	APP *int `json:"app,omitempty"`

	// Utilize default or Pay-As-You-Go billing.
	// +kubebuilder:validation:Enum:=default;metered
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Billing",order=8
	Billing string `json:"billing,omitempty"`

	// Installation token that prevents unauthorized hosts from being accidentally or maliciously added to your customer ID (CID).
	// +kubebuilder:validation:Pattern:="^[0-9a-fA-F]{8}$"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Provisioning Token",order=2
	PToken string `json:"provisioning_token,omitempty"`

	// Sensor grouping tags are optional, user-defined identifiers that can used to group and filter hosts. Allowed characters: all alphanumerics, '/', '-', and '_'.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Sensor Grouping Tags",order=6
	Tags []string `json:"tags,omitempty"`

	// Set sensor trace level.
	// +kubebuilder:validation:Enum:=none;err;warn;info;debug
	// +kubebuilder:default:=none
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Trace Level",order=7
	Trace string `json:"trace,omitempty"`
}
