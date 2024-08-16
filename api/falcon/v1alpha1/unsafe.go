package v1alpha1

// FalconUnsafe configures various options that go against industry practices or are otherwise not recommended for use.
// Adjusting these settings may result in incorrect or undesirable behavior. Proceed at your own risk.
// For more information, please see https://github.com/CrowdStrike/falcon-operator/blob/main/UNSAFE.md.
type FalconUnsafe struct {
	// UpdatePolicy is the name of a sensor update policy configured and enabled in Falcon UI. It is ignored when Image and/or Version are set.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Admission Controller Update Policy",order=1
	UpdatePolicy *string `json:"updatePolicy,omitempty"`
}
