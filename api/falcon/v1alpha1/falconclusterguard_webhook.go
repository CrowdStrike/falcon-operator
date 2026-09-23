package v1alpha1

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

//+kubebuilder:webhook:path=/validate-falcon-crowdstrike-com-v1alpha1-falconclusterguard,mutating=false,failurePolicy=ignore,sideEffects=None,groups=falcon.crowdstrike.com,resources=falconclusterguards,verbs=create;update,versions=v1alpha1,name=vfalconclusterguard.kb.io,admissionReviewVersions=v1

// FalconClusterGuardValidator enforces that FalconClusterGuard is managed by a FalconDeployment
// and that at least one component is enabled.
type FalconClusterGuardValidator struct{}

var _ admission.Validator[*FalconClusterGuard] = &FalconClusterGuardValidator{}

func (v *FalconClusterGuardValidator) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &FalconClusterGuard{}).
		WithValidator(v).
		Complete()
}

func (v *FalconClusterGuardValidator) ValidateCreate(_ context.Context, obj *FalconClusterGuard) (admission.Warnings, error) {
	var warnings admission.Warnings
	if !hasFalconDeploymentOwner(obj) {
		warnings = append(warnings, "FalconClusterGuard is not managed by a FalconDeployment; if you are deploying more than one component, using a FalconDeployment is recommended")
	}
	w, err := validateAtLeastOneEnabled(obj)
	return append(warnings, w...), err
}

func (v *FalconClusterGuardValidator) ValidateUpdate(_ context.Context, _, newObj *FalconClusterGuard) (admission.Warnings, error) {
	return validateAtLeastOneEnabled(newObj)
}

func (v *FalconClusterGuardValidator) ValidateDelete(_ context.Context, _ *FalconClusterGuard) (admission.Warnings, error) {
	return nil, nil
}

func hasFalconDeploymentOwner(obj *FalconClusterGuard) bool {
	for _, ref := range obj.GetOwnerReferences() {
		if ref.Kind == "FalconDeployment" {
			return true
		}
	}
	return false
}

func validateAtLeastOneEnabled(obj *FalconClusterGuard) (admission.Warnings, error) {
	if !obj.Spec.AdmissionConfig.IsEnabled() && !obj.Spec.NodeSensor.IsEnabled() {
		return nil, fmt.Errorf("at least one of admissionConfig.enabled or nodeSensor.enabled must be true")
	}
	return nil, nil
}
