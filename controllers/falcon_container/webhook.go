package falcon

import (
	"context"
	"fmt"
	"reflect"

	"github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/go-logr/logr"
	arv1 "k8s.io/api/admissionregistration/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	webhookName = "mutatingwebhook.sidecar.falcon.crowdstrike.com"
)

func (r *FalconContainerReconciler) reconcileWebhook(ctx context.Context, log logr.Logger, falconContainer *v1alpha1.FalconContainer, caBundle []byte) (*arv1.MutatingWebhookConfiguration, error) {
	disableDefaultNSInjection := false

	if falconContainer.Spec.Injector.DisableDefaultNSInjection {
		disableDefaultNSInjection = falconContainer.Spec.Injector.DisableDefaultNSInjection
	}

	webhook := r.newWebhook(webhookName, caBundle, disableDefaultNSInjection, falconContainer)
	existingWebhook := &arv1.MutatingWebhookConfiguration{}

	err := r.Client.Get(ctx, types.NamespacedName{Name: webhookName}, existingWebhook)
	if err != nil {
		if errors.IsNotFound(err) {
			if err = ctrl.SetControllerReference(falconContainer, webhook, r.Scheme); err != nil {
				return &arv1.MutatingWebhookConfiguration{}, fmt.Errorf("unable to set controller reference on mutating webhook configuration %s: %v", webhook.ObjectMeta.Name, err)
			}

			return webhook, r.Create(ctx, log, falconContainer, webhook)
		}

		return &arv1.MutatingWebhookConfiguration{}, fmt.Errorf("unable to query existing mutating webhook configuration %s: %v", webhookName, err)
	}

	if !reflect.DeepEqual(webhook.Webhooks[0], existingWebhook.Webhooks[0]) {
		existingWebhook.Webhooks[0] = webhook.Webhooks[0]

		return webhook, r.Update(ctx, log, falconContainer, existingWebhook)
	}

	return existingWebhook, nil

}
func (r *FalconContainerReconciler) newWebhook(webhookName string, caBundle []byte, disableNSInjection bool, falconContainer *v1alpha1.FalconContainer) *arv1.MutatingWebhookConfiguration {
	sideEffects := arv1.SideEffectClassNone
	reinvocationPolicy := arv1.NeverReinvocationPolicy
	failurePolicy := arv1.Fail
	matchPolicy := arv1.Equivalent
	scope := arv1.AllScopes
	var timeoutSeconds int32 = 30
	path := "/mutate"
	operatorSelector := metav1.LabelSelectorOpNotIn
	operatorValues := []string{"disabled"}

	if disableNSInjection {
		operatorSelector = metav1.LabelSelectorOpIn
		operatorValues = []string{"enabled"}
	}

	return &arv1.MutatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "MutatingWebhookConfiguration",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      webhookName,
			Namespace: r.Namespace(),
			Labels:    FcLabels,
		},
		Webhooks: []arv1.MutatingWebhook{
			{
				Name:                    webhookName,
				AdmissionReviewVersions: common.FCAdmissionReviewVersions(),
				SideEffects:             &sideEffects,
				FailurePolicy:           &failurePolicy,
				ReinvocationPolicy:      &reinvocationPolicy,
				ObjectSelector:          &metav1.LabelSelector{},
				MatchPolicy:             &matchPolicy,
				ClientConfig: arv1.WebhookClientConfig{
					CABundle: caBundle,
					Service: &arv1.ServiceReference{
						Name:      injectorName,
						Namespace: r.Namespace(),
						Path:      &path,
						Port:      falconContainer.Spec.Injector.ListenPort,
					},
				},
				TimeoutSeconds: &timeoutSeconds,
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
