package v1alpha1

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

//+kubebuilder:webhook:path=/validate-falcon-crowdstrike-com-v1alpha1-falcondeployment,mutating=false,failurePolicy=ignore,sideEffects=None,groups=falcon.crowdstrike.com,resources=falcondeployments,verbs=create;update,versions=v1alpha1,name=vfalcondeployment.kb.io,admissionReviewVersions=v1

// FalconDeploymentValidator enforces deprecation rules on FalconDeployment.
// deployNodeSensor and deployAdmissionController may not be enabled on create,
// and may not be changed from false to true on update.
type FalconDeploymentValidator struct{}

var _ admission.Validator[*FalconDeployment] = &FalconDeploymentValidator{}

func (v *FalconDeploymentValidator) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &FalconDeployment{}).
		WithValidator(v).
		Complete()
}

func (v *FalconDeploymentValidator) ValidateCreate(_ context.Context, obj *FalconDeployment) (admission.Warnings, error) {
	if obj.Spec.DeployNodeSensor != nil && *obj.Spec.DeployNodeSensor {
		return nil, fmt.Errorf("deployNodeSensor is not supported; use FalconClusterGuard instead")
	}
	if obj.Spec.DeployAdmissionController != nil && *obj.Spec.DeployAdmissionController {
		return nil, fmt.Errorf("deployAdmissionController is not supported; use FalconClusterGuard instead")
	}
	return nil, nil
}

func (v *FalconDeploymentValidator) ValidateUpdate(_ context.Context, oldObj, newObj *FalconDeployment) (admission.Warnings, error) {
	oldNodeSensor := oldObj.Spec.DeployNodeSensor != nil && *oldObj.Spec.DeployNodeSensor
	newNodeSensor := newObj.Spec.DeployNodeSensor != nil && *newObj.Spec.DeployNodeSensor
	if !oldNodeSensor && newNodeSensor {
		return nil, fmt.Errorf("deployNodeSensor cannot be enabled; use FalconClusterGuard instead")
	}

	oldAdmission := oldObj.Spec.DeployAdmissionController != nil && *oldObj.Spec.DeployAdmissionController
	newAdmission := newObj.Spec.DeployAdmissionController != nil && *newObj.Spec.DeployAdmissionController
	if !oldAdmission && newAdmission {
		return nil, fmt.Errorf("deployAdmissionController cannot be enabled; use FalconClusterGuard instead")
	}

	return nil, nil
}

func (v *FalconDeploymentValidator) ValidateDelete(_ context.Context, _ *FalconDeployment) (admission.Warnings, error) {
	return nil, nil
}
