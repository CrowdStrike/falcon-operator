package falcon

import (
	"context"
	"os"
	"testing"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVersionLock_WithAutoUpdateDisabled(t *testing.T) {
	reconciler := &FalconContainerReconciler{}
	container := &falconv1alpha1.FalconContainer{}
	container.Status.Sensor = stringPointer("some sensor")
	container.Spec.Advanced.AutoUpdate = stringPointer(falconv1alpha1.Off)
	assert.True(t, reconciler.versionLock(container))
}

func TestVersionLock_WithForcedAutoUpdate(t *testing.T) {
	reconciler := &FalconContainerReconciler{}
	container := &falconv1alpha1.FalconContainer{}
	container.Status.Sensor = stringPointer("some sensor")
	container.Spec.Advanced.AutoUpdate = stringPointer(falconv1alpha1.Force)
	assert.False(t, reconciler.versionLock(container))
}

func TestVersionLock_WithNormalAutoUpdate(t *testing.T) {
	reconciler := &FalconContainerReconciler{}
	container := &falconv1alpha1.FalconContainer{}
	container.Status.Sensor = stringPointer("some sensor")
	container.Spec.Advanced.AutoUpdate = stringPointer(falconv1alpha1.Normal)
	assert.False(t, reconciler.versionLock(container))
}

func TestVersionLock_WithBlankUpdatePolicy(t *testing.T) {
	reconciler := &FalconContainerReconciler{}
	container := &falconv1alpha1.FalconContainer{}
	container.Status.Sensor = stringPointer("some sensor")
	container.Spec.Advanced.UpdatePolicy = stringPointer("")
	assert.True(t, reconciler.versionLock(container))
}

func TestVersionLock_WithDifferentVersion(t *testing.T) {
	reconciler := &FalconContainerReconciler{}
	container := &falconv1alpha1.FalconContainer{}
	container.Status.Sensor = stringPointer("some sensor")
	container.Spec.Version = stringPointer("different version")
	assert.False(t, reconciler.versionLock(container))
}

func TestVersionLock_WithLatestVersion(t *testing.T) {
	reconciler := &FalconContainerReconciler{}
	container := &falconv1alpha1.FalconContainer{}
	container.Status.Sensor = stringPointer("some sensor")
	assert.True(t, reconciler.versionLock(container))
}

func TestVersionLock_WithNoCurrentSensor(t *testing.T) {
	reconciler := &FalconContainerReconciler{}
	container := &falconv1alpha1.FalconContainer{}
	assert.False(t, reconciler.versionLock(container))
}

func TestVersionLock_WithSameVersion(t *testing.T) {
	reconciler := &FalconContainerReconciler{}
	container := &falconv1alpha1.FalconContainer{}
	container.Status.Sensor = stringPointer("some sensor")
	container.Spec.Version = container.Status.Sensor
	assert.True(t, reconciler.versionLock(container))
}

func TestVersionLock_WithUpdatePolicy(t *testing.T) {
	reconciler := &FalconContainerReconciler{}
	container := &falconv1alpha1.FalconContainer{}
	container.Status.Sensor = stringPointer("some sensor")
	container.Spec.Advanced.UpdatePolicy = stringPointer("some policy")
	assert.False(t, reconciler.versionLock(container))
}

func TestImageUri_SpecImageOverridesRelatedImage(t *testing.T) {
	require.NoError(t, os.Setenv("RELATED_IMAGE_SIDECAR_SENSOR", "bundled-image:latest"))
	t.Cleanup(func() { os.Unsetenv("RELATED_IMAGE_SIDECAR_SENSOR") })

	r := &FalconContainerReconciler{}
	container := &falconv1alpha1.FalconContainer{}
	container.Spec.Image = stringPointer("my-registry/falcon-sensor:1.0")

	uri, err := r.imageUri(context.Background(), container)
	require.NoError(t, err)
	assert.Equal(t, "my-registry/falcon-sensor:1.0", uri)
}

func TestImageUri_RelatedImage_NoCredentials(t *testing.T) {
	require.NoError(t, os.Setenv("RELATED_IMAGE_SIDECAR_SENSOR", "bundled-image:latest"))
	t.Cleanup(func() { os.Unsetenv("RELATED_IMAGE_SIDECAR_SENSOR") })

	r := &FalconContainerReconciler{}
	container := &falconv1alpha1.FalconContainer{}
	container.Spec.FalconAPI = &falconv1alpha1.FalconAPI{}

	uri, err := r.imageUri(context.Background(), container)
	require.NoError(t, err)
	assert.Equal(t, "bundled-image:latest", uri)
}

func TestImageUri_RelatedImage_ClientIdOnly_UsesRelatedImage(t *testing.T) {
	require.NoError(t, os.Setenv("RELATED_IMAGE_SIDECAR_SENSOR", "bundled-image:latest"))
	t.Cleanup(func() { os.Unsetenv("RELATED_IMAGE_SIDECAR_SENSOR") })

	r := &FalconContainerReconciler{}
	container := &falconv1alpha1.FalconContainer{}
	container.Spec.FalconAPI = &falconv1alpha1.FalconAPI{ClientId: "abc123"}

	uri, err := r.imageUri(context.Background(), container)
	require.NoError(t, err)
	assert.Equal(t, "bundled-image:latest", uri)
}

func TestImageUri_RelatedImage_WithCredentials_FallsThrough(t *testing.T) {
	require.NoError(t, os.Setenv("RELATED_IMAGE_SIDECAR_SENSOR", "bundled-image:latest"))
	t.Cleanup(func() { os.Unsetenv("RELATED_IMAGE_SIDECAR_SENSOR") })

	r := &FalconContainerReconciler{}
	container := &falconv1alpha1.FalconContainer{}
	container.Spec.FalconAPI = &falconv1alpha1.FalconAPI{ClientId: "abc123", ClientSecret: "def456"}

	_, err := r.imageUri(context.Background(), container)
	assert.Error(t, err)
}

func TestRegistryUri_CrowdStrike_NoCredentials(t *testing.T) {
	r := &FalconContainerReconciler{}
	container := &falconv1alpha1.FalconContainer{}
	container.Spec.Registry.Type = falconv1alpha1.RegistryTypeCrowdStrike

	_, err := r.registryUri(context.Background(), container)
	assert.ErrorContains(t, err, "must be configured when using CrowdStrike registry")
}

func TestImageUri_RelatedImage_WithFalconSecret_FallsThrough(t *testing.T) {
	require.NoError(t, os.Setenv("RELATED_IMAGE_SIDECAR_SENSOR", "bundled-image:latest"))
	t.Cleanup(func() { os.Unsetenv("RELATED_IMAGE_SIDECAR_SENSOR") })

	r := &FalconContainerReconciler{}
	container := &falconv1alpha1.FalconContainer{}
	container.Spec.FalconAPI = &falconv1alpha1.FalconAPI{}
	container.Spec.FalconSecret = falconv1alpha1.FalconSecret{Enabled: true}

	_, err := r.imageUri(context.Background(), container)
	assert.Error(t, err)
}

func stringPointer(s string) *string {
	return &s
}
