package controllers

import (
	"testing"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/stretchr/testify/assert"
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

func stringPointer(s string) *string {
	return &s
}
