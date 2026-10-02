package v1alpha1

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

//+kubebuilder:webhook:path=/validate-falcon-crowdstrike-com-v1alpha1-falconadmission,mutating=false,failurePolicy=ignore,sideEffects=None,groups=falcon.crowdstrike.com,resources=falconadmissions,verbs=create;update,versions=v1alpha1,name=vfalconadmission.kb.io,admissionReviewVersions=v1

// FalconAdmissionValidator blocks creation of new FalconAdmission objects.
// FalconAdmission is deprecated; use FalconDeployment instead.
type FalconAdmissionValidator struct{}

var _ admission.Validator[*FalconAdmission] = &FalconAdmissionValidator{}

func (v *FalconAdmissionValidator) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &FalconAdmission{}).
		WithValidator(v).
		Complete()
}

func (v *FalconAdmissionValidator) ValidateCreate(_ context.Context, _ *FalconAdmission) (admission.Warnings, error) {
	return nil, fmt.Errorf("FalconAdmission is deprecated; use FalconDeployment instead")
}

func (v *FalconAdmissionValidator) ValidateUpdate(_ context.Context, _, _ *FalconAdmission) (admission.Warnings, error) {
	return nil, fmt.Errorf("FalconAdmission is deprecated; use FalconDeployment instead")
}

func (v *FalconAdmissionValidator) ValidateDelete(_ context.Context, _ *FalconAdmission) (admission.Warnings, error) {
	return nil, nil
}
