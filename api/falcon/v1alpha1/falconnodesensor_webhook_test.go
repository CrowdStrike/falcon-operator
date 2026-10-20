package v1alpha1

import (
	"context"
	"testing"
)

func TestFalconNodeSensorValidator_ValidateCreate(t *testing.T) {
	v := &FalconNodeSensorValidator{}
	_, err := v.ValidateCreate(context.Background(), &FalconNodeSensor{})
	if err == nil {
		t.Error("ValidateCreate: expected error for deprecated FalconNodeSensor, got nil")
	}
}
