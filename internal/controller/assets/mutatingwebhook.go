package assets

import (
	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	arv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// MutatingWebhook creates a MutatingWebhookConfiguration object
func MutatingWebhook(name string, namespace string, webhookName string, caBundle []byte, disableNSInjection bool, falconContainer *falconv1alpha1.FalconContainer) *arv1.MutatingWebhookConfiguration {
	sideEffects := arv1.SideEffectClassNone
	reinvocationPolicy := arv1.NeverReinvocationPolicy
	failurePolicy := arv1.Fail
	matchPolicy := arv1.Equivalent
	scope := arv1.AllScopes
	var timeoutSeconds int32 = 30
	path := "/mutate"
	operatorSelector := metav1.LabelSelectorOpNotIn
	operatorValues := []string{"disabled"}
	labels := common.CRLabels("mutatingwebhook", name, common.FalconSidecarSensor)

	if disableNSInjection {
		operatorSelector = metav1.LabelSelectorOpIn
		operatorValues = []string{"enabled"}
	}

	return &arv1.MutatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: arv1.SchemeGroupVersion.String(),
			Kind:       "MutatingWebhookConfiguration",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      webhookName,
			Namespace: namespace,
			Labels:    labels,
		},
		Webhooks: []arv1.MutatingWebhook{
			{
				Name:                    webhookName,
				AdmissionReviewVersions: []string{"v1"},
				SideEffects:             &sideEffects,
				FailurePolicy:           &failurePolicy,
				ReinvocationPolicy:      &reinvocationPolicy,
				MatchPolicy:             &matchPolicy,
				ClientConfig: arv1.WebhookClientConfig{
					CABundle: caBundle,
					Service: &arv1.ServiceReference{
						Name:      name,
						Namespace: namespace,
						Path:      &path,
						Port:      falconContainer.Spec.Injector.ListenPort,
					},
				},
				TimeoutSeconds: &timeoutSeconds,
				ObjectSelector: &metav1.LabelSelector{},
				NamespaceSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      common.FalconContainerInjection,
							Operator: operatorSelector,
							Values:   operatorValues,
						},
						{
							Key:      "control-plane",
							Operator: metav1.LabelSelectorOpDoesNotExist,
						},
					},
				},
				Rules: []arv1.RuleWithOperations{
					{
						Operations: []arv1.OperationType{arv1.Create},
						Rule: arv1.Rule{
							APIGroups:   []string{""},
							APIVersions: []string{"v1"},
							Resources:   []string{"pods"},
							Scope:       &scope,
						},
					},
				},
			},
		},
	}
}
