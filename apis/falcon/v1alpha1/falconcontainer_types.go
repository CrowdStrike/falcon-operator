package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// FalconAPI configures connection from your local Falcon operator to CrowdStrike Falcon platform.
type FalconAPI struct {
	// CloudRegion defines CrowdStrike Falcon Cloud Region to which the operator will connect to
	// +kubebuilder:validation:Enum=autodiscover;us-1;us-2;eu-1;us-gov-1
	// +kubebuilder:default=autodiscover
	CloudRegion string `json:"cloud_region"`
	// Falcon OAuth2 API Client ID
	ClientId string `json:"client_id"`
	// Falcon OAuth2 API Client Secret
	ClientSecret string `json:"client_secret"`
	// Falcon Customer ID (CID) Override (optional, default is derived from the API Key pair)
	// +kubebuilder:validation:Pattern="^[0-9a-fA-F]{32}-[0-9a-fA-F]{2}$"
	CID *string `json:"cid,omitempty"`
}

// RegistryTLSSpec configures TLS for registry pushing
type RegistryTLSSpec struct {
	// Allow pushing to docker registries over HTTPS with failed TLS verification. Note that this does not affect other TLS connections.
	InsecureSkipVerify bool `json:"insecure_skip_verify,omitempty"`
}

type RegistryTypeSpec string

const (
	// RegistryTypeOpenshift represents OpenShift Image Stream
	RegistryTypeOpenshift RegistryTypeSpec = "openshift"
	// RegistryTypeGCR represents Google Container Registry
	RegistryTypeGCR RegistryTypeSpec = "gcr"
	// RegistryTypeECR represents AWS Elastic Container Registry
	RegistryTypeECR RegistryTypeSpec = "ecr"
	// RegistryTypeACR represents Azure Container Registry
	RegistryTypeACR RegistryTypeSpec = "acr"
	// RegistryTypeCrowdStrike represents deployment that won't push Falcon Container to local registry, instead CrowdStrike registry will be used.
	RegistryTypeCrowdStrike RegistryTypeSpec = "crowdstrike"
)

// RegistrySpec configures container image registry to which the Falcon Container image will be pushed
type RegistrySpec struct {
	// Type of the registry to be used
	// +kubebuilder:validation:Enum=acr;ecr;gcr;crowdstrike;openshift
	Type RegistryTypeSpec `json:"type"`

	// TLS configures TLS connection for push of Falcon Container image to the registry
	TLS RegistryTLSSpec `json:"tls,omitempty"`
	// Azure Container Registry Name represents the name of the ACR for the Falcon Container push. Only applicable to Azure cloud.
	AcrName *string `json:"acr_name,omitempty"`
}

// FalconContainerSpec defines the desired state of FalconContainer
type FalconContainerSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// FalconAPI configures connection from your local Falcon operator to CrowdStrike Falcon platform.
	FalconAPI FalconAPI `json:"falcon_api"`
	// Registry configures container image registry to which the Falcon Container image will be pushed
	Registry RegistrySpec `json:"registry,omitempty"`

	FalconContainerSensor       `json:"falcon,omitempty"`
	FalconContainerSensorConfig `json:"container,omitempty"`

	// InstallerArgs are passed directly down to the Falcon Container Installer. Users are advised to consult Falcon Container documentation to learn about available command line arguments at https://falcon.crowdstrike.com/documentation/146/falcon-container-sensor-for-linux
	InstallerArgs []string `json:"installer_args,omitempty"`
	// Falcon Container Version. The latest version will be selected when version specifier is missing.
	Version *string `json:"version,omitempty"`
}

// FalconContainerSensorConfig defines aspects about how the Falcon Container sensor and injector work.
// +k8s:openapi-gen=true
type FalconContainerSensorConfig struct {
	// Image Pull Policy
	// +kubebuilder:default=Always
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// Name of the Falcon Sensor container to pull. Format should be repository/namespace/name:tag
	// +kubebuilder:default="falcon-sensor:latest"
	Image string `json:"image,omitempty"`

	// Kills pod after a specificed amount of time (in seconds). Default is 30 seconds.
	// +kubebuilder:default=30
	TimeoutSeconds int32 `json:"timeout,omitempty"`

	// Configure number of replicas for the injector deployment
	// +kubebuilder:default=1
	Replicas int32 `json:"replicas"`

	// Configure the requests and limits of the injector. Please note that setting limits can have an affect on how effective the sensor is depending on cluster load.
	InjectorResources Resources `json:"resources,omitempty"`

	// Use URL instead of a service definition for the MutatingWebhook configuration
	// +kubebuilder:default=""
	URL string `json:"url,omitempty"`

	// Configure the list of namespaces that should have access to pull the Falcon sensor from a registry that requires authentication.
	Namespaces []string `json:"namespaces,omitempty"`

	// Set to true if connecting to a registry that requires authentication
	// +kubebuilder:default=false
	EnablePullSecrets bool `json:"enablePullSecrets,omitempty"`

	// Custom registry pull secret (in base64) to pull authenticate to an external registry.
	// +kubebuilder:default=""
	ContainerRegistryPullSecret string `json:"registryPullSecret,omitempty"`

	// Enable if running in Azure AKS
	// +kubebuilder:default=false
	AzureEnable bool `json:"azure"`

	// Path to the Kubernetes Azure config file on worker nodes (e.g. /etc/kubernetes/azure.json)
	// +kubebuilder:default="/etc/kubernetes/azure.json"
	AzureConfig string `json:"azureConfig"`

	// Port Falcon Injector listens on
	// +kubebuilder:default=4433
	InjectorPort int32 `json:"injectorPort"`

	// Disable injection for all Namespaces
	// +kubebuilder:default=false
	DisableNSInjection bool `json:"disableNSInjection"`

	// Disable injection for all Pods
	// +kubebuilder:default=false
	DisablePodInjection bool `json:"disablePodInjection"`

	// Certificate validity duration in number of days
	// +kubebuilder:default=3650
	CertExpiration int `json:"certExpiration"`

	// The path which the container runtime daemon socket can be found on worker node. This will mount the socket in the sensor container
	ContainerDaemonSocket string `json:"containerDaemonSocket,omitempty"`
}

// Configure the Resources of the injector pod
// +k8s:openapi-gen=true
type Resources struct {
	Requests InjectorRequests `json:"requests"`
	Limits   Limits           `json:"limits,omitempty"`
}

// Configure the resources requests of the injector pod
// +k8s:openapi-gen=true
type InjectorRequests struct {
	// +kubebuilder:default:="10m"
	CPU string `json:"cpu"`
	// +kubebuilder:default:="20Mi"
	Memory string `json:"memory"`
}

// Configure the resource limits of the injector pod or container sidecar sensor
// +k8s:openapi-gen=true
type Limits struct {
	CPU    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
}

// CrowdStrike Falcon Sensor configuration settings.
// +k8s:openapi-gen=true
type FalconContainerSensor struct {
	// Falcon Customer ID (CID)
	CID string `json:"cid,omitempty"`
	// Enable the App Proxy Port (APP). Uncommon in container-based deployments.
	APD bool `json:"apd,omitempty"`
	// App Proxy Hostname (APH). Uncommon in container-based deployments.
	APH string `json:"aph,omitempty"`
	// App Proxy Port (APP). Uncommon in container-based deployments.
	APP string `json:"app,omitempty"`
	// Utilize default or metered billing.
	Billing bool `json:"billing,omitempty"`
	// Provisioning token.
	PToken string `json:"provisioning_token,omitempty"`
	// List of tags for sensor grouping. Allowed characters: all alphanumerics, '/', '-', and '_'.
	Tags []string `json:"tags,omitempty"`
	// Set trace level. Options are [none|err|warn|info|debug].
	Trace string `json:"trace,omitempty"`
	// Configure the requests and limits of the container sidecar sensor. Please note that setting limits can have an affect on how effective the sensor is depending on cluster load.
	ContainerResources *ContainerResources `json:"resources,omitempty"`
}

// Configure the resources requests of the container sidecar sensor
// +k8s:openapi-gen=true
type ContainerResources struct {
	Requests *ContainerRequests `json:"requests,omitempty"`
	Limits   *Limits            `json:"limits,omitempty"`
}

// Configure the resources requests of the container sidecar sensor
// +k8s:openapi-gen=true
type ContainerRequests struct {
	CPU    string `json:"cpu,omitempty"`
	Memory string `json:"memory,omitempty"`
}

// Represents the status of Falcon deployment
type FalconContainerStatusPhase string

const (
	// PhasePending represents the deployment to be started
	PhasePending FalconContainerStatusPhase = "PENDING"
	// PhaseBuilding represents the deployment before the falcon image is successfully fetched
	PhaseBuilding FalconContainerStatusPhase = "BUILDING"
	// PhaseConfiguring represents the state when injector/installer is being run
	PhaseConfiguring FalconContainerStatusPhase = "CONFIGURING"
	// PhaseDeploying represents the state when injector is being deployed on the cluster
	PhaseDeploying FalconContainerStatusPhase = "DEPLOYING"
	// PhaseValidating represents the state when falcon-operator validates injector pod installation
	PhaseValidating FalconContainerStatusPhase = "VALIDATING"
	// PhaseDone represents the Falcon Protection being successfully installed
	PhaseDone FalconContainerStatusPhase = "DONE"
)

// FalconContainerStatus defines the observed state of FalconContainer
type FalconContainerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Phase or the status of the deployment
	Phase FalconContainerStatusPhase `json:"phase,omitempty"`

	// ErrorMessage informs user of the last notable error. Users are welcomed to see the operator logs
	// to understand the full context.
	ErrorMessage string `json:"errormsg,omitempty"`

	Version *string `json:"version,omitempty"`

	// RetryAttempt is number of previous failed attempts. Valid values: 0-5
	RetryAttempt *uint8 `json:"retry_attempt,omitempty"`
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:scope=Cluster
//+kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase",description="Phase of deployment"
//+kubebuilder:printcolumn:name="Version",type="string",JSONPath=".status.version",description="Version of Falcon Container"
//+kubebuilder:printcolumn:name="Error",type="string",JSONPath=".status.errormsg",description="Last error message"

// FalconContainer is the Schema for the falconcontainers API
type FalconContainer struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FalconContainerSpec   `json:"spec,omitempty"`
	Status FalconContainerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// FalconContainerList contains a list of FalconContainer
type FalconContainerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FalconContainer `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FalconContainer{}, &FalconContainerList{})
}
