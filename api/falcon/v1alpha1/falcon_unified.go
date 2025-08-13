package v1alpha1

// FalconUnified Sensor configuration settings, extends FalconSensor with fields used for unified installation
type FalconUnified struct {
	FalconSensor `json:",inline"`

	// Falcon Customer Cloud Region - With the unified installer, you can let the sensor discover the CID's cloud automatically, or you can specify the cloud where the CID resides.
	// +kubebuilder:validation:Enum=us-1;us-2;eu-1;us-gov-1;us-gov-2
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="CrowdStrike Falcon Cloud Region"
	Cloud string `json:"cloud,omitempty"`
}
