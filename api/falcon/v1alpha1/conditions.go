package v1alpha1

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
	ReasonFailed           string = "Failed"
	ReasonDiscovered       string = "Discovered"
)
