package assets

import (
	"testing"

	"github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/google/go-cmp/cmp"
	"golang.org/x/exp/maps"
	arv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestValidatingWebhook tests the ValidatingWebhook function
func TestValidatingWebhook(t *testing.T) {
	want := testValidatingWebhook("test", "test", "test", []byte("test"), 123, arv1.Ignore, []string{"ns1", "ns2"})

	got := ValidatingWebhook("test", "test", "test", []byte("test"), 123, arv1.Ignore, []string{"ns1", "ns2"})
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("ValidatingWebhook() mismatch (-want +got): %s", diff)
	}
}

// testValidatingWebhook is a helper function to create a ValidatingWebhookConfiguration
func testValidatingWebhook(name string, namespace string, webhookName string, caBundle []byte, port int32, failPolicy arv1.FailurePolicyType, disabledNamespaces []string) *arv1.ValidatingWebhookConfiguration {
	failurePolicy := arv1.Ignore
	matchPolicy := arv1.Equivalent
	sideEffects := arv1.SideEffectClassNone
	timeoutSeconds := int32(10)
	operatorSelector := metav1.LabelSelectorOpNotIn
	path := "/validate"
	scope := arv1.AllScopes
	admissionOperatorValues := []string{"disabled"}
	labels := common.CRLabels("validatingwebhook", name, common.FalconAdmissionController)
	helmLabels := map[string]string{
		"app":                         "falcon-kac",
		"app.kubernetes.io/name":      "falcon-kac",
		"app.kubernetes.io/component": "kac",
	}
	maps.Copy(labels, helmLabels)

	return &arv1.ValidatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: arv1.SchemeGroupVersion.String(),
			Kind:       "ValidatingWebhookConfiguration",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      labels,
			Annotations: map[string]string{"admissions.enforcer/disabled": "true"},
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
							Values:   []string{"ns1", "ns2"},
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
