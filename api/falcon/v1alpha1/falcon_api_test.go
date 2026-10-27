package v1alpha1

import (
	"context"
	"testing"

	internalErrors "github.com/crowdstrike/falcon-operator/internal/errors"
	"github.com/stretchr/testify/assert"
)

func TestIsConfigured_NoCrendentials(t *testing.T) {
	fa := &FalconAPI{}
	assert.False(t, fa.IsConfigured(FalconSecret{}))
}

func TestIsConfigured_ClientIdOnly(t *testing.T) {
	fa := &FalconAPI{ClientId: "abc123"}
	assert.False(t, fa.IsConfigured(FalconSecret{}))
}

func TestIsConfigured_ClientSecretOnly(t *testing.T) {
	fa := &FalconAPI{ClientSecret: "secret"}
	assert.False(t, fa.IsConfigured(FalconSecret{}))
}

func TestIsConfigured_BothClientFields(t *testing.T) {
	fa := &FalconAPI{ClientId: "abc123", ClientSecret: "secret"}
	assert.True(t, fa.IsConfigured(FalconSecret{}))
}

func TestIsConfigured_FalconSecretEnabled(t *testing.T) {
	fa := &FalconAPI{}
	assert.True(t, fa.IsConfigured(FalconSecret{Enabled: true}))
}

func TestIsConfigured_FalconSecretEnabledWithCredentials(t *testing.T) {
	fa := &FalconAPI{ClientId: "abc123"}
	assert.True(t, fa.IsConfigured(FalconSecret{Enabled: true}))
}

func TestIsConfigured_NilReceiver_NoSecret(t *testing.T) {
	var fa *FalconAPI
	assert.False(t, fa.IsConfigured(FalconSecret{}))
}

func TestIsConfigured_NilReceiver_FalconSecretEnabled(t *testing.T) {
	var fa *FalconAPI
	assert.True(t, fa.IsConfigured(FalconSecret{Enabled: true}))
}

func TestApiConfigWithSecret_NilReceiver_NoSecret(t *testing.T) {
	var fa *FalconAPI
	cfg, err := fa.ApiConfigWithSecret(context.Background(), nil, FalconSecret{})
	assert.ErrorIs(t, err, internalErrors.ErrNilFalconAPIConfiguration)
	assert.NotNil(t, cfg)
}

func TestApiConfigWithSecret_NoSecret(t *testing.T) {
	fa := &FalconAPI{ClientId: "abc123", ClientSecret: "secret", CloudRegion: "us-1"}
	cfg, err := fa.ApiConfigWithSecret(context.Background(), nil, FalconSecret{})
	assert.NoError(t, err)
	assert.Equal(t, "abc123", cfg.ClientId)
	assert.Equal(t, "secret", cfg.ClientSecret)
}
