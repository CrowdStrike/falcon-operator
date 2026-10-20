package v1alpha1

import (
	"context"
	"testing"
)

func TestFalconImageAnalyzerValidator_ValidateCreate(t *testing.T) {
	v := &FalconImageAnalyzerValidator{}
	_, err := v.ValidateCreate(context.Background(), &FalconImageAnalyzer{})
	if err == nil {
		t.Error("ValidateCreate: expected error for deprecated FalconImageAnalyzer, got nil")
	}
}

func TestFalconImageAnalyzerValidator_ValidateUpdate(t *testing.T) {
	v := &FalconImageAnalyzerValidator{}
	_, err := v.ValidateUpdate(context.Background(), &FalconImageAnalyzer{}, &FalconImageAnalyzer{})
	if err == nil {
		t.Error("ValidateUpdate: expected error for deprecated FalconImageAnalyzer, got nil")
	}
}
