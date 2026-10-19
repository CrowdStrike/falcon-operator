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

func TestFalconNodeSensorValidator_ValidateUpdate(t *testing.T) {
	v := &FalconNodeSensorValidator{}
	_, err := v.ValidateUpdate(context.Background(), &FalconNodeSensor{}, &FalconNodeSensor{})
	if err != nil {
		t.Errorf("ValidateUpdate: expected nil, got %v", err)
	}
}

func TestFalconNodeSensorValidator_ValidateDelete(t *testing.T) {
	v := &FalconNodeSensorValidator{}
	_, err := v.ValidateDelete(context.Background(), &FalconNodeSensor{})
	if err != nil {
		t.Errorf("ValidateDelete: expected nil, got %v", err)
	}
}
