package falcon

import (
	"testing"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestVersionLock_WithDifferentVersion(t *testing.T) {
	reconciler := &FalconContainerReconciler{}
	admission := &falconv1alpha1.FalconContainer{}
	admission.Status.Sensor = stringPointer("some sensor")
	admission.Spec.Version = stringPointer("different version")
	assert.False(t, reconciler.versionLock(admission))
}

func TestVersionLock_WithLatestVersion(t *testing.T) {
	reconciler := &FalconContainerReconciler{}
	admission := &falconv1alpha1.FalconContainer{}
	admission.Status.Sensor = stringPointer("some sensor")
	assert.True(t, reconciler.versionLock(admission))
}

func TestVersionLock_WithNoCurrentSensor(t *testing.T) {
	reconciler := &FalconContainerReconciler{}
	admission := &falconv1alpha1.FalconContainer{}
	assert.False(t, reconciler.versionLock(admission))
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
	container.Spec.UpdatePolicy = stringPointer("some policy")
	assert.False(t, reconciler.versionLock(container))
}

func stringPointer(s string) *string {
	return &s
}
