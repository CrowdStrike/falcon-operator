package v1alpha1

import (
	"context"
	"testing"
)

func TestFalconAdmissionValidator_ValidateCreate(t *testing.T) {
	v := &FalconAdmissionValidator{}
	_, err := v.ValidateCreate(context.Background(), &FalconAdmission{})
	if err == nil {
		t.Error("ValidateCreate: expected error for deprecated FalconAdmission, got nil")
	}
}

func TestFalconAdmissionValidator_ValidateUpdate(t *testing.T) {
	v := &FalconAdmissionValidator{}
	_, err := v.ValidateUpdate(context.Background(), &FalconAdmission{}, &FalconAdmission{})
	if err == nil {
		t.Error("ValidateUpdate: expected error for deprecated FalconAdmission, got nil")
	}
}

func TestFalconAdmissionValidator_ValidateDelete(t *testing.T) {
	v := &FalconAdmissionValidator{}
	_, err := v.ValidateDelete(context.Background(), &FalconAdmission{})
	if err != nil {
		t.Errorf("ValidateDelete: expected nil, got %v", err)
	}
}
