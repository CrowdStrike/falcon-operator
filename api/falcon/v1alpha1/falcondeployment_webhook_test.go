package v1alpha1

import (
	"context"
	"testing"
)

func boolPtr(b bool) *bool { return &b }

func TestFalconDeploymentValidator_ValidateCreate(t *testing.T) {
	v := &FalconDeploymentValidator{}

	tests := []struct {
		name    string
		spec    FalconDeploymentSpec
		wantErr bool
	}{
		{
			name:    "both nil — allowed",
			spec:    FalconDeploymentSpec{},
			wantErr: false,
		},
		{
			name:    "both false — allowed",
			spec:    FalconDeploymentSpec{DeployNodeSensor: boolPtr(false), DeployAdmissionController: boolPtr(false)},
			wantErr: false,
		},
		{
			name:    "deployNodeSensor true — rejected",
			spec:    FalconDeploymentSpec{DeployNodeSensor: boolPtr(true)},
			wantErr: true,
		},
		{
			name:    "deployAdmissionController true — rejected",
			spec:    FalconDeploymentSpec{DeployAdmissionController: boolPtr(true)},
			wantErr: true,
		},
		{
			name:    "both true — rejected",
			spec:    FalconDeploymentSpec{DeployNodeSensor: boolPtr(true), DeployAdmissionController: boolPtr(true)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			obj := &FalconDeployment{Spec: tt.spec}
			_, err := v.ValidateCreate(context.Background(), obj)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateCreate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFalconDeploymentValidator_ValidateUpdate(t *testing.T) {
	v := &FalconDeploymentValidator{}

	tests := []struct {
		name    string
		old     FalconDeploymentSpec
		new     FalconDeploymentSpec
		wantErr bool
	}{
		{
			name:    "no change, both false — allowed",
			old:     FalconDeploymentSpec{DeployNodeSensor: boolPtr(false), DeployAdmissionController: boolPtr(false)},
			new:     FalconDeploymentSpec{DeployNodeSensor: boolPtr(false), DeployAdmissionController: boolPtr(false)},
			wantErr: false,
		},
		{
			name:    "no change, both nil — allowed",
			old:     FalconDeploymentSpec{},
			new:     FalconDeploymentSpec{},
			wantErr: false,
		},
		{
			name:    "deployNodeSensor false → true — rejected",
			old:     FalconDeploymentSpec{DeployNodeSensor: boolPtr(false)},
			new:     FalconDeploymentSpec{DeployNodeSensor: boolPtr(true)},
			wantErr: true,
		},
		{
			name:    "deployNodeSensor nil → true — rejected",
			old:     FalconDeploymentSpec{},
			new:     FalconDeploymentSpec{DeployNodeSensor: boolPtr(true)},
			wantErr: true,
		},
		{
			name:    "deployAdmissionController false → true — rejected",
			old:     FalconDeploymentSpec{DeployAdmissionController: boolPtr(false)},
			new:     FalconDeploymentSpec{DeployAdmissionController: boolPtr(true)},
			wantErr: true,
		},
		{
			name:    "deployAdmissionController nil → true — rejected",
			old:     FalconDeploymentSpec{},
			new:     FalconDeploymentSpec{DeployAdmissionController: boolPtr(true)},
			wantErr: true,
		},
		{
			name:    "deployNodeSensor true → false — allowed",
			old:     FalconDeploymentSpec{DeployNodeSensor: boolPtr(true)},
			new:     FalconDeploymentSpec{DeployNodeSensor: boolPtr(false)},
			wantErr: false,
		},
		{
			name:    "deployAdmissionController true → false — allowed",
			old:     FalconDeploymentSpec{DeployAdmissionController: boolPtr(true)},
			new:     FalconDeploymentSpec{DeployAdmissionController: boolPtr(false)},
			wantErr: false,
		},
		{
			name:    "both fields false → true simultaneously — rejected",
			old:     FalconDeploymentSpec{DeployNodeSensor: boolPtr(false), DeployAdmissionController: boolPtr(false)},
			new:     FalconDeploymentSpec{DeployNodeSensor: boolPtr(true), DeployAdmissionController: boolPtr(true)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			oldObj := &FalconDeployment{Spec: tt.old}
			newObj := &FalconDeployment{Spec: tt.new}
			_, err := v.ValidateUpdate(context.Background(), oldObj, newObj)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateUpdate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFalconDeploymentValidator_ValidateDelete(t *testing.T) {
	v := &FalconDeploymentValidator{}
	_, err := v.ValidateDelete(context.Background(), &FalconDeployment{})
	if err != nil {
		t.Errorf("ValidateDelete: expected nil, got %v", err)
	}
}
