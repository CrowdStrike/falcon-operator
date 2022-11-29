package v1alpha1

import (
	"context"
	"fmt"

	"github.com/crowdstrike/falcon-operator/pkg/falcon_api"
	"github.com/crowdstrike/falcon-operator/version"
	"github.com/crowdstrike/gofalcon/falcon"
)

// FalconAPI configures connection from your local Falcon operator to CrowdStrike Falcon platform.
type FalconAPI struct {
	// Cloud Region defines CrowdStrike Falcon Cloud Region to which the operator will connect and register.
	// +kubebuilder:validation:Enum=autodiscover;us-1;us-2;eu-1;us-gov-1
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="CrowdStrike Falcon Cloud Region",order=3
	CloudRegion string `json:"cloud_region"`
	// Falcon OAuth2 API Client ID
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Client ID",order=1
	ClientId string `json:"client_id"`
	// Falcon OAuth2 API Client Secret
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Client Secret",order=2
	ClientSecret string `json:"client_secret"`
	// Falcon Customer ID (CID) Override (optional, default is derived from the API Key pair)
	// +kubebuilder:validation:Pattern="^[0-9a-fA-F]{32}-[0-9a-fA-F]{2}$"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Customer ID (CID)",order=4
	CID *string `json:"cid,omitempty"`
}

// CrowdStrike Falcon Sensor configuration settings.
// +k8s:openapi-gen=true
type FalconSensor struct {
	// Falcon Customer ID (CID)
	// +kubebuilder:validation:Pattern:="^[0-9a-fA-F]{32}-[0-9a-fA-F]{2}$"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Customer ID (CID)",order=1
	CID *string `json:"cid,omitempty"`
	// Disable the Falcon Sensor's use of a proxy.
	// +kubebuilder:default:=false
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Disable Falcon Proxy",order=3
	APD *bool `json:"apd,omitempty"`
	// The application proxy host to use for Falcon sensor proxy configuration.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Disable Falcon Proxy Host",order=4
	APH string `json:"aph,omitempty"`
	// The application proxy port to use for Falcon sensor proxy configuration.
	// +kubebuilder:validation:Minimum:=0
	// +kubebuilder:validation:Maximum:=65535
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Proxy Port",order=5
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

// RegistryTLSSpec configures TLS for registry pushing
type RegistryTLSSpec struct {
	// Allow pushing to docker registries over HTTPS with failed TLS verification. Note that this does not affect other TLS connections.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Skip Registry TLS Verification",order=1
	InsecureSkipVerify bool `json:"insecure_skip_verify,omitempty"`
	// Allow for users to provide a CA Cert Bundle, as either a string or base64 encoded string
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Registry CA Certificate Bundle; optionally (double) base64 encoded",order=2
	CACertificate string `json:"caCertificate,omitempty"`
	// Allow for users to provide a ConfigMap containing a CA Cert Bundle under a key ending in .crt
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="ConfigMap containing Registry CA Certificate Bundle",order=3
	CACertificateConfigMap string `json:"caCertificateConfigMap,omitempty"`
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

// ApiConfig generates standard gofalcon library api config
func (fa *FalconAPI) ApiConfig() *falcon.ApiConfig {
	return &falcon.ApiConfig{
		Cloud:             falcon.Cloud(fa.CloudRegion),
		ClientId:          fa.ClientId,
		ClientSecret:      fa.ClientSecret,
		UserAgentOverride: fmt.Sprintf("falcon-operator/%s", version.Version),
	}
}

func (fa *FalconAPI) FalconCloud(ctx context.Context) (falcon.CloudType, error) {
	return falcon_api.FalconCloud(ctx, fa.ApiConfig())
}
