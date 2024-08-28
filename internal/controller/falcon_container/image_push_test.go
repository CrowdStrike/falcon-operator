package falcon

import (
	"testing"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestVersionLock_WithAutoUpdateDisabled(t *testing.T) {
	reconciler := &FalconContainerReconciler{}
	container := &falconv1alpha1.FalconContainer{}
	container.Status.Sensor = stringPointer("some sensor")
	container.Spec.Unsafe.AutoUpdate = stringPointer(falconv1alpha1.Off)
	assert.True(t, reconciler.versionLock(container))
}

func TestVersionLock_WithForcedAutoUpdate(t *testing.T) {
	reconciler := &FalconContainerReconciler{}
	container := &falconv1alpha1.FalconContainer{}
	container.Status.Sensor = stringPointer("some sensor")
	container.Spec.Unsafe.AutoUpdate = stringPointer(falconv1alpha1.Force)
	assert.False(t, reconciler.versionLock(container))
}

func TestVersionLock_WithNormalAutoUpdate(t *testing.T) {
	reconciler := &FalconContainerReconciler{}
	container := &falconv1alpha1.FalconContainer{}
	container.Status.Sensor = stringPointer("some sensor")
	container.Spec.Unsafe.AutoUpdate = stringPointer(falconv1alpha1.Normal)
	assert.False(t, reconciler.versionLock(container))
}

func TestVersionLock_WithBlankUpdatePolicy(t *testing.T) {
	reconciler := &FalconContainerReconciler{}
	container := &falconv1alpha1.FalconContainer{}
	container.Status.Sensor = stringPointer("some sensor")
	container.Spec.Unsafe.UpdatePolicy = stringPointer("")
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
	container.Spec.Unsafe.UpdatePolicy = stringPointer("some policy")
	assert.False(t, reconciler.versionLock(container))
}

func boolPointer(b bool) *bool {
	return &b
}

func stringPointer(s string) *string {
	return &s
}
