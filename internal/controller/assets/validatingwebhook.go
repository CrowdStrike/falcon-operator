package assets

import (
	"github.com/crowdstrike/falcon-operator/pkg/common"
	arv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ValidatingWebhook returns a ValidatingWebhookConfiguration object
func ValidatingWebhook(name string, namespace string, webhookName string, caBundle []byte, port int32, failPolicy arv1.FailurePolicyType, disabledNamespaces []string) *arv1.ValidatingWebhookConfiguration {
	failurePolicy := arv1.Ignore
	matchPolicy := arv1.Equivalent
	sideEffects := arv1.SideEffectClassNone
	timeoutSeconds := int32(5)
	operatorSelector := metav1.LabelSelectorOpNotIn
	path := "/validate"
	scope := arv1.AllScopes
	admissionOperatorValues := []string{"disabled"}
	labels := common.CRLabels("validatingwebhook", name, common.FalconAdmissionController)

	return &arv1.ValidatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: arv1.SchemeGroupVersion.String(),
			Kind:       "ValidatingWebhookConfiguration",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    labels,
		},
		Webhooks: []arv1.ValidatingWebhook{
			{
				Name:                    webhookName,
				AdmissionReviewVersions: []string{"v1"},
				SideEffects:             &sideEffects,
				FailurePolicy:           &failPolicy,
				MatchPolicy:             &matchPolicy,
				ClientConfig: arv1.WebhookClientConfig{
					CABundle: caBundle,
					Service: &arv1.ServiceReference{
						Name:      name,
						Namespace: namespace,
						Path:      &path,
						Port:      &port,
					},
				},
				TimeoutSeconds: &timeoutSeconds,
				NamespaceSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "kubernetes.io/metadata.name",
							Operator: operatorSelector,
							Values:   disabledNamespaces,
						},
						{
							Key:      common.FalconAdmissionReviewKey,
							Operator: operatorSelector,
							Values:   admissionOperatorValues,
						},
					},
				},
				Rules: []arv1.RuleWithOperations{
					{
						Operations: []arv1.OperationType{
							arv1.Create,
							arv1.Update,
						},
						Rule: arv1.Rule{
							APIGroups: []string{
								"",
							},
							APIVersions: []string{
								"v1",
							},
							Resources: []string{
								"pods",
								"pods/ephemeralcontainers",
							},
							Scope: &scope,
						},
					},
				},
			},
			{
				Name:                    "workload." + webhookName,
				AdmissionReviewVersions: []string{"v1"},
				SideEffects:             &sideEffects,
				FailurePolicy:           &failurePolicy,
				MatchPolicy:             &matchPolicy,
				ClientConfig: arv1.WebhookClientConfig{
					CABundle: caBundle,
					Service: &arv1.ServiceReference{
						Name:      name,
						Namespace: namespace,
						Path:      &path,
						Port:      &port,
					},
				},
				TimeoutSeconds: &timeoutSeconds,
				NamespaceSelector: &metav1.LabelSelector{
					MatchExpressions: []metav1.LabelSelectorRequirement{
						{
							Key:      "kubernetes.io/metadata.name",
							Operator: operatorSelector,
							Values:   disabledNamespaces,
						},
						{
							Key:      common.FalconAdmissionReviewKey,
							Operator: operatorSelector,
							Values:   admissionOperatorValues,
						},
					},
				},
				Rules: []arv1.RuleWithOperations{
					{
						Operations: []arv1.OperationType{
							arv1.Create,
							arv1.Update,
						},
						Rule: arv1.Rule{
							APIGroups: []string{
								"",
							},
							APIVersions: []string{
								"v1",
							},
							Resources: []string{
								"replicationcontrollers",
								"services",
							},
							Scope: &scope,
						},
					},
					{
						Operations: []arv1.OperationType{
							arv1.Create,
							arv1.Update,
						},
						Rule: arv1.Rule{
							APIGroups: []string{
								"apps",
							},
							APIVersions: []string{
								"v1",
							},
							Resources: []string{
								"daemonsets",
								"deployments",
								"replicasets",
								"statefulsets",
							},
							Scope: &scope,
						},
					},
					{
						Operations: []arv1.OperationType{
							arv1.Create,
							arv1.Update,
						},
						Rule: arv1.Rule{
							APIGroups: []string{
								"batch",
							},
							APIVersions: []string{
								"v1",
							},
							Resources: []string{
								"cronjobs",
								"jobs",
							},
							Scope: &scope,
						},
					},
				},
			},
		},
	}
}
