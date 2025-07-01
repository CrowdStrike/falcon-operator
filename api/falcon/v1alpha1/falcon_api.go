package v1alpha1

import (
	"context"
	"fmt"
	"strings"

	internalErrors "github.com/crowdstrike/falcon-operator/internal/errors"
	"github.com/crowdstrike/falcon-operator/pkg/falcon_api"
	"github.com/crowdstrike/falcon-operator/pkg/falcon_secret"
	"github.com/crowdstrike/falcon-operator/version"

	"github.com/crowdstrike/gofalcon/falcon"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// FalconAPI configures connection from your local Falcon operator to CrowdStrike Falcon platform.
type FalconAPI struct {
	// Cloud Region defines CrowdStrike Falcon Cloud Region to which the operator will connect and register.
	// +kubebuilder:validation:Enum=autodiscover;us-1;us-2;eu-1;us-gov-1;us-gov-2
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="CrowdStrike Falcon Cloud Region",order=3
	CloudRegion string `json:"cloud_region"`

	// Falcon OAuth2 API Client ID
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Client ID",order=1,xDescriptors="urn:alm:descriptor:com.tectonic.ui:password"
	ClientId string `json:"client_id,omitempty"`

	// Falcon OAuth2 API Client Secret
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Client Secret",order=2,xDescriptors="urn:alm:descriptor:com.tectonic.ui:password"
	ClientSecret string `json:"client_secret,omitempty"`

	// Falcon Customer ID (CID) Override (optional, default is derived from the API Key pair)
	// +kubebuilder:validation:Pattern="^[0-9a-fA-F]{32}-[0-9a-fA-F]{2}$"
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Falcon Customer ID (CID)",order=4
	CID *string `json:"cid,omitempty"`

	// Specifies the hostname of the API endpoint to use. If blank, the public Falcon API endpoint is used.
	// Intentionally not exported as a resource property.
	HostOverride string `json:"-"`
}

// RegistryTLSSpec configures TLS for registry pushing
type RegistryTLSSpec struct {
	// Allow pushing to docker registries over HTTPS with failed TLS verification. Note that this does not affect other TLS connections.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Skip Registry TLS Verification",order=1,xDescriptors="urn:alm:descriptor:com.tectonic.ui:booleanSwitch"
	InsecureSkipVerify bool `json:"insecure_skip_verify,omitempty"`

	// Allow for users to provide a CA Cert Bundle, as either a string or base64 encoded string
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Registry CA Certificate Bundle; optionally (double) base64 encoded",order=2
	CACertificate string `json:"caCertificate,omitempty"`

	// Allow for users to provide a ConfigMap containing a CA Cert Bundle under a key ending in .crt
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="ConfigMap containing Registry CA Certificate Bundle",order=3,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:selector:core:v1:ConfigMap"}
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
	// Type of container registry to be used
	// +kubebuilder:validation:Enum=acr;ecr;gcr;crowdstrike;openshift
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Registry Type",order=1
	Type RegistryTypeSpec `json:"type"`

	// TLS configures TLS connection for push of Falcon Container image to the registry
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Registry TLS Configuration",order=2
	TLS RegistryTLSSpec `json:"tls,omitempty"`

	// Azure Container Registry Name represents the name of the ACR for the Falcon Container push. Only applicable to Azure cloud.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Azure Container Registry Name",order=3
	AcrName *string `json:"acr_name,omitempty"`
}

// ApiConfig generates standard gofalcon library api config
func (fa *FalconAPI) ApiConfig() *falcon.ApiConfig {
	return &falcon.ApiConfig{
		Cloud:             falcon.Cloud(fa.CloudRegion),
		ClientId:          fa.ClientId,
		ClientSecret:      fa.ClientSecret,
		HostOverride:      fa.HostOverride,
		UserAgentOverride: fmt.Sprintf("falcon-operator/%s", version.Version),
	}
}

// ApiConfigWithSecret generates standard gofalcon library api config, with sensitive data injected via a k8s secret
func (fa *FalconAPI) ApiConfigWithSecret(
	ctx context.Context,
	k8sReader client.Reader,
	falconSecret FalconSecret,
) (*falcon.ApiConfig, error) {
	if !falconSecret.Enabled {
		return fa.ApiConfig(), nil
	}

	falconApiSecret := &corev1.Secret{}
	falconSecretNamespacedName := types.NamespacedName{
		Name:      falconSecret.SecretName,
		Namespace: falconSecret.Namespace,
	}

	err := k8sReader.Get(ctx, falconSecretNamespacedName, falconApiSecret)
	if err != nil {
		return &falcon.ApiConfig{}, err
	}

	clientId, clientSecret := falcon_secret.GetFalconCredsFromSecret(falconApiSecret)
	if strings.TrimSpace(clientId) == "" || strings.TrimSpace(clientSecret) == "" {
		return &falcon.ApiConfig{}, internalErrors.ErrMissingFalconAPICredentialsInSecret
	}

	cloudRegion := ""
	hostOverride := ""
	if fa != nil {
		cloudRegion = fa.CloudRegion
		hostOverride = fa.HostOverride
	}

	return &falcon.ApiConfig{
		Cloud:             falcon.Cloud(cloudRegion),
		ClientId:          clientId,
		ClientSecret:      clientSecret,
		HostOverride:      hostOverride,
		UserAgentOverride: fmt.Sprintf("falcon-operator/%s", version.Version),
	}, nil
}

func (fa *FalconAPI) FalconCloudWithSecret(
	ctx context.Context,
	k8sReader client.Reader,
	falconSecret FalconSecret,
) (falcon.CloudType, error) {
	falconApiConfig, err := fa.ApiConfigWithSecret(ctx, k8sReader, falconSecret)
	if err != nil {
		return falconApiConfig.Cloud, err
	}

	return falcon_api.FalconCloud(ctx, falconApiConfig)
}
