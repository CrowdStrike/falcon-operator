package v1alpha1

// FalconInternal defines configurations used for internal testing
type FalconInternal struct {
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Custom sensor image repository path within registry.crowdstrike.com",order=6
	CrowdstrikeRegistryRepoOverride *string `json:"crowdstrikeRegistryRepoOverride,omitempty"`
}
