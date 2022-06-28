package v1alpha1

import (
	"context"

	"github.com/crowdstrike/falcon-operator/pkg/falcon_api"
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

// ApiConfig generates standard gofalcon library api config
func (fa *FalconAPI) ApiConfig() *falcon.ApiConfig {
	return &falcon.ApiConfig{
		Cloud:        falcon.Cloud(fa.CloudRegion),
		ClientId:     fa.ClientId,
		ClientSecret: fa.ClientSecret,
	}
}

func (fa *FalconAPI) FalconCloud(ctx context.Context) (falcon.CloudType, error) {
	return falcon_api.FalconCloud(ctx, fa.ApiConfig())
}
