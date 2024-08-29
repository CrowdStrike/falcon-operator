package v1alpha1

import "strings"

const (
	Force  = "force"
	Normal = "normal"
	Off    = "off"
)

// FalconAdvanced configures various options that go against industry practices or are otherwise not recommended for use.
// Adjusting these settings may result in incorrect or undesirable behavior. Proceed at your own risk.
// For more information, please see https://github.com/CrowdStrike/falcon-operator/blob/main/docs/ADVANCED.md.
type FalconAdvanced struct {
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

func (advanced FalconAdvanced) GetUpdatePolicy() string {
	if advanced.UpdatePolicy == nil {
		return ""
	}

	return strings.TrimSpace(*advanced.UpdatePolicy)
}

func (advanced FalconAdvanced) HasUpdatePolicy() bool {
	return advanced.GetUpdatePolicy() != ""
}

func (advanced FalconAdvanced) IsAutoUpdating() bool {
	if advanced.AutoUpdate == nil {
		return false
	}

	return *advanced.AutoUpdate != "off"
}

func (advanced FalconAdvanced) IsAutoUpdatingForced() bool {
	if advanced.AutoUpdate == nil {
		return false
	}

	return *advanced.AutoUpdate == "force"
}
