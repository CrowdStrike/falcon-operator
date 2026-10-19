package v1alpha1

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

//+kubebuilder:webhook:path=/validate-falcon-crowdstrike-com-v1alpha1-falconnodesensor,mutating=false,failurePolicy=ignore,sideEffects=None,groups=falcon.crowdstrike.com,resources=falconnodesensors,verbs=create,versions=v1alpha1,name=vfalconnodesensor.kb.io,admissionReviewVersions=v1

// FalconNodeSensorValidator blocks creation of new FalconNodeSensor objects.
// FalconNodeSensor is deprecated; use FalconDeployment instead.
type FalconNodeSensorValidator struct{}

var _ admission.Validator[*FalconNodeSensor] = &FalconNodeSensorValidator{}

func (v *FalconNodeSensorValidator) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &FalconNodeSensor{}).
		WithValidator(v).
		Complete()
}

func (v *FalconNodeSensorValidator) ValidateCreate(_ context.Context, _ *FalconNodeSensor) (admission.Warnings, error) {
	return nil, fmt.Errorf("FalconNodeSensor is deprecated; use FalconDeployment or FalconClusterGuard instead")
}

func (v *FalconNodeSensorValidator) ValidateUpdate(_ context.Context, _, _ *FalconNodeSensor) (admission.Warnings, error) {
	return nil, nil
}

func (v *FalconNodeSensorValidator) ValidateDelete(_ context.Context, _ *FalconNodeSensor) (admission.Warnings, error) {
	return nil, nil
}
