package v1alpha1

import (
	"context"
	"fmt"

	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

//+kubebuilder:webhook:path=/validate-falcon-crowdstrike-com-v1alpha1-falconimageanalyzer,mutating=false,failurePolicy=ignore,sideEffects=None,groups=falcon.crowdstrike.com,resources=falconimageanalyzers,verbs=create;update,versions=v1alpha1,name=vfalconimageanalyzer.kb.io,admissionReviewVersions=v1

// FalconImageAnalyzerValidator blocks creation of new FalconImageAnalyzer objects.
// FalconImageAnalyzer is deprecated; use FalconClusterGuard instead.
type FalconImageAnalyzerValidator struct{}

var _ admission.Validator[*FalconImageAnalyzer] = &FalconImageAnalyzerValidator{}

func (v *FalconImageAnalyzerValidator) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr, &FalconImageAnalyzer{}).
		WithValidator(v).
		Complete()
}

func (v *FalconImageAnalyzerValidator) ValidateCreate(_ context.Context, _ *FalconImageAnalyzer) (admission.Warnings, error) {
	return nil, fmt.Errorf("FalconImageAnalyzer is deprecated; use FalconClusterGuard instead")
}

func (v *FalconImageAnalyzerValidator) ValidateUpdate(_ context.Context, _, _ *FalconImageAnalyzer) (admission.Warnings, error) {
	return nil, fmt.Errorf("FalconImageAnalyzer is deprecated; use FalconClusterGuard instead")
}

func (v *FalconImageAnalyzerValidator) ValidateDelete(_ context.Context, _ *FalconImageAnalyzer) (admission.Warnings, error) {
	return nil, nil
}
