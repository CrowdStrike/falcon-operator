package v1alpha1

import "strings"

const (
	Force  = "force"
	Normal = "normal"
	Off    = "off"
)

// FalconUnsafe configures various options that go against industry practices or are otherwise not recommended for use.
// Adjusting these settings may result in incorrect or undesirable behavior. Proceed at your own risk.
// For more information, please see https://github.com/CrowdStrike/falcon-operator/blob/main/UNSAFE.md.
type FalconUnsafe struct {
	// UpdatePolicy is the name of a sensor update policy configured and enabled in Falcon UI. It is ignored when Image and/or Version are set.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Sensor Update Policy",order=1
	UpdatePolicy *string `json:"updatePolicy,omitempty"`

	// AutoUpdate determines whether to install new versions of the sensor as they become available. Defaults to "off" and is ignored if FalconAPI is not set.
	// Setting this to "force" causes the reconciler to run on every polling cycle, even if a new sensor version is not available.
	// Setting it to "normal" only reconciles when a new version is detected.
	// +kubebuilder:validation:Enum=off;normal;force
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Sensor Automatic Updates",order=2
	AutoUpdate *string `json:"autoUpdate,omitempty"`
}

func (notSafe FalconUnsafe) GetUpdatePolicy() string {
	if notSafe.UpdatePolicy == nil {
		return ""
	}

	return strings.TrimSpace(*notSafe.UpdatePolicy)
}

func (notSafe FalconUnsafe) HasUpdatePolicy() bool {
	return notSafe.GetUpdatePolicy() != ""
}

func (notSafe FalconUnsafe) IsAutoUpdating() bool {
	if notSafe.AutoUpdate == nil {
		return false
	}

	return *notSafe.AutoUpdate != "off"
}

func (notSafe FalconUnsafe) IsAutoUpdatingForced() bool {
	if notSafe.AutoUpdate == nil {
		return false
	}

	return *notSafe.AutoUpdate == "force"
}
