package controllers

import (
	"context"
	"os"
	"testing"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionLock_WithDifferentVersion(t *testing.T) {
	reconciler := &FalconAdmissionReconciler{}
	admission := &falconv1alpha1.FalconAdmission{}
	admission.Status.Sensor = stringPointer("some sensor")
	admission.Spec.Version = stringPointer("different version")
	assert.False(t, reconciler.versionLock(admission))
}

func TestVersionLock_WithLatestVersion(t *testing.T) {
	reconciler := &FalconAdmissionReconciler{}
	admission := &falconv1alpha1.FalconAdmission{}
	admission.Status.Sensor = stringPointer("some sensor")
	assert.True(t, reconciler.versionLock(admission))
}

func TestVersionLock_WithNoCurrentSensor(t *testing.T) {
	reconciler := &FalconAdmissionReconciler{}
	admission := &falconv1alpha1.FalconAdmission{}
	assert.False(t, reconciler.versionLock(admission))
}

func TestVersionLock_WithSameVersion(t *testing.T) {
	reconciler := &FalconAdmissionReconciler{}
	admission := &falconv1alpha1.FalconAdmission{}
	admission.Status.Sensor = stringPointer("some sensor")
	admission.Spec.Version = admission.Status.Sensor
	assert.True(t, reconciler.versionLock(admission))
}

func TestImageUri_SpecImageOverridesRelatedImage(t *testing.T) {
	require.NoError(t, os.Setenv("RELATED_IMAGE_ADMISSION_CONTROLLER", "bundled-image:latest"))
	t.Cleanup(func() { os.Unsetenv("RELATED_IMAGE_ADMISSION_CONTROLLER") })

	r := &FalconAdmissionReconciler{}
	admission := &falconv1alpha1.FalconAdmission{}
	admission.Spec.Image = "my-registry/falcon-kac:1.0"

	uri, err := r.imageUri(context.Background(), admission)
	require.NoError(t, err)
	assert.Equal(t, "my-registry/falcon-kac:1.0", uri)
}

func TestImageUri_RelatedImage_NoCredentials(t *testing.T) {
	require.NoError(t, os.Setenv("RELATED_IMAGE_ADMISSION_CONTROLLER", "bundled-image:latest"))
	t.Cleanup(func() { os.Unsetenv("RELATED_IMAGE_ADMISSION_CONTROLLER") })

	r := &FalconAdmissionReconciler{}
	admission := &falconv1alpha1.FalconAdmission{}
	admission.Spec.FalconAPI = &falconv1alpha1.FalconAPI{}

	uri, err := r.imageUri(context.Background(), admission)
	require.NoError(t, err)
	assert.Equal(t, "bundled-image:latest", uri)
}

func TestImageUri_RelatedImage_ClientIdOnly_UsesRelatedImage(t *testing.T) {
	require.NoError(t, os.Setenv("RELATED_IMAGE_ADMISSION_CONTROLLER", "bundled-image:latest"))
	t.Cleanup(func() { os.Unsetenv("RELATED_IMAGE_ADMISSION_CONTROLLER") })

	r := &FalconAdmissionReconciler{}
	admission := &falconv1alpha1.FalconAdmission{}
	admission.Spec.FalconAPI = &falconv1alpha1.FalconAPI{ClientId: "abc123"}

	uri, err := r.imageUri(context.Background(), admission)
	require.NoError(t, err)
	assert.Equal(t, "bundled-image:latest", uri)
}

func TestImageUri_RelatedImage_WithCredentials_FallsThrough(t *testing.T) {
	require.NoError(t, os.Setenv("RELATED_IMAGE_ADMISSION_CONTROLLER", "bundled-image:latest"))
	t.Cleanup(func() { os.Unsetenv("RELATED_IMAGE_ADMISSION_CONTROLLER") })

	r := &FalconAdmissionReconciler{}
	admission := &falconv1alpha1.FalconAdmission{}
	admission.Spec.FalconAPI = &falconv1alpha1.FalconAPI{ClientId: "abc123", ClientSecret: "def456"}

	// Should not return the bundled image; falls through to registry path which
	// errors without a real client — that confirms it didn't short-circuit.
	_, err := r.imageUri(context.Background(), admission)
	assert.Error(t, err)
}

func TestRegistryUri_CrowdStrike_NoCredentials(t *testing.T) {
	r := &FalconAdmissionReconciler{}
	admission := &falconv1alpha1.FalconAdmission{}
	admission.Spec.Registry.Type = falconv1alpha1.RegistryTypeCrowdStrike

	_, err := r.registryUri(context.Background(), admission)
	assert.ErrorContains(t, err, "must be configured when using CrowdStrike registry")
}

func TestImageUri_RelatedImage_WithFalconSecret_FallsThrough(t *testing.T) {
	require.NoError(t, os.Setenv("RELATED_IMAGE_ADMISSION_CONTROLLER", "bundled-image:latest"))
	t.Cleanup(func() { os.Unsetenv("RELATED_IMAGE_ADMISSION_CONTROLLER") })

	r := &FalconAdmissionReconciler{}
	admission := &falconv1alpha1.FalconAdmission{}
	admission.Spec.FalconAPI = &falconv1alpha1.FalconAPI{}
	admission.Spec.FalconSecret = falconv1alpha1.FalconSecret{Enabled: true}

	_, err := r.imageUri(context.Background(), admission)
	assert.Error(t, err)
}

func stringPointer(s string) *string {
	return &s
}
