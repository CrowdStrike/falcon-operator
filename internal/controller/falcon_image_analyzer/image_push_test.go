package falcon

import (
	"context"
	"os"
	"testing"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestImageUri_SpecImageOverridesRelatedImage(t *testing.T) {
	require.NoError(t, os.Setenv("RELATED_IMAGE_IMAGE_ANALYZER", "bundled-image:latest"))
	t.Cleanup(func() { os.Unsetenv("RELATED_IMAGE_IMAGE_ANALYZER") })

	r := &FalconImageAnalyzerReconciler{}
	fia := &falconv1alpha1.FalconImageAnalyzer{}
	fia.Spec.Image = "my-registry/falcon-imageanalyzer:1.0"

	uri, err := r.imageUri(context.Background(), fia)
	require.NoError(t, err)
	assert.Equal(t, "my-registry/falcon-imageanalyzer:1.0", uri)
}

func TestImageUri_RelatedImage_NoCredentials(t *testing.T) {
	require.NoError(t, os.Setenv("RELATED_IMAGE_IMAGE_ANALYZER", "bundled-image:latest"))
	t.Cleanup(func() { os.Unsetenv("RELATED_IMAGE_IMAGE_ANALYZER") })

	r := &FalconImageAnalyzerReconciler{}
	fia := &falconv1alpha1.FalconImageAnalyzer{}
	fia.Spec.FalconAPI = &falconv1alpha1.FalconAPI{}

	uri, err := r.imageUri(context.Background(), fia)
	require.NoError(t, err)
	assert.Equal(t, "bundled-image:latest", uri)
}

func TestImageUri_RelatedImage_ClientIdOnly_UsesRelatedImage(t *testing.T) {
	require.NoError(t, os.Setenv("RELATED_IMAGE_IMAGE_ANALYZER", "bundled-image:latest"))
	t.Cleanup(func() { os.Unsetenv("RELATED_IMAGE_IMAGE_ANALYZER") })

	r := &FalconImageAnalyzerReconciler{}
	fia := &falconv1alpha1.FalconImageAnalyzer{}
	fia.Spec.FalconAPI = &falconv1alpha1.FalconAPI{ClientId: "abc123"}

	uri, err := r.imageUri(context.Background(), fia)
	require.NoError(t, err)
	assert.Equal(t, "bundled-image:latest", uri)
}

func TestImageUri_RelatedImage_WithCredentials_FallsThrough(t *testing.T) {
	require.NoError(t, os.Setenv("RELATED_IMAGE_IMAGE_ANALYZER", "bundled-image:latest"))
	t.Cleanup(func() { os.Unsetenv("RELATED_IMAGE_IMAGE_ANALYZER") })

	r := &FalconImageAnalyzerReconciler{}
	fia := &falconv1alpha1.FalconImageAnalyzer{}
	fia.Spec.FalconAPI = &falconv1alpha1.FalconAPI{ClientId: "abc123", ClientSecret: "def456"}

	_, err := r.imageUri(context.Background(), fia)
	assert.Error(t, err)
}

func TestRegistryUri_CrowdStrike_NoCredentials(t *testing.T) {
	r := &FalconImageAnalyzerReconciler{}
	fia := &falconv1alpha1.FalconImageAnalyzer{}
	fia.Spec.Registry.Type = falconv1alpha1.RegistryTypeCrowdStrike

	_, err := r.registryUri(context.Background(), fia)
	assert.ErrorContains(t, err, "must be configured when using CrowdStrike registry")
}

func TestImageUri_RelatedImage_WithFalconSecret_FallsThrough(t *testing.T) {
	require.NoError(t, os.Setenv("RELATED_IMAGE_IMAGE_ANALYZER", "bundled-image:latest"))
	t.Cleanup(func() { os.Unsetenv("RELATED_IMAGE_IMAGE_ANALYZER") })

	r := &FalconImageAnalyzerReconciler{}
	fia := &falconv1alpha1.FalconImageAnalyzer{}
	fia.Spec.FalconAPI = &falconv1alpha1.FalconAPI{}
	fia.Spec.FalconSecret = falconv1alpha1.FalconSecret{Enabled: true}

	_, err := r.imageUri(context.Background(), fia)
	assert.Error(t, err)
}
