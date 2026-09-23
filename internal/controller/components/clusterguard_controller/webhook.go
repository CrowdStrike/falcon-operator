package clusterguard_controller

import (
	"context"
	"reflect"
	"sort"

	k8sutils "github.com/crowdstrike/falcon-operator/internal/controller/common"
	pkgcommon "github.com/crowdstrike/falcon-operator/pkg/common"
	arv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
)

// ValidatingWebhook builds the ValidatingWebhookConfiguration for FalconClusterGuard
// with three webhooks: pod admission, workload admission, and a test webhook.
func (a *ClusterGuardController) ValidatingWebhook(caBundle []byte) *arv1.ValidatingWebhookConfiguration {
	namespace := a.cfg.InstallNamespace
	extraDisabledNamespaces := a.cfg.ClusterGuardControllerConfig.DisabledNamespaces.Namespaces
	failurePolicy := arv1.Ignore
	if a.cfg.ClusterGuardControllerConfig.FailurePolicy != "" {
		failurePolicy = a.cfg.ClusterGuardControllerConfig.FailurePolicy
	}
	matchPolicy := arv1.Equivalent
	sideEffects := arv1.SideEffectClassNone
	timeoutSeconds := int32(10)
	excludeOp := metav1.LabelSelectorOpNotIn
	scope := arv1.AllScopes
	webhookName := pkgcommon.AdmissionValidatingWebhookName
	path := "/validate"
	port := int32(443)
	if a.cfg.ClusterGuardControllerConfig.Port != nil {
		port = *a.cfg.ClusterGuardControllerConfig.Port
	}

	excludedNamespaces := append(pkgcommon.DefaultDisabledNamespaces,
		namespace,
		"falcon-system",
		"falcon-kubernetes-protection",
	)
	excludedNamespaces = append(excludedNamespaces, extraDisabledNamespaces...)
	// deduplicate
	seen := map[string]struct{}{}
	unique := excludedNamespaces[:0]
	for _, ns := range excludedNamespaces {
		if _, ok := seen[ns]; !ok {
			seen[ns] = struct{}{}
			unique = append(unique, ns)
		}
	}
	excludedNamespaces = unique
	sort.Strings(excludedNamespaces)

	namespaceSelector := &metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{
				Key:      "kubernetes.io/metadata.name",
				Operator: excludeOp,
				Values:   excludedNamespaces,
			},
			{
				Key:      "falcon-kac.crowdstrike.com/admission-review",
				Operator: metav1.LabelSelectorOpNotIn,
				Values:   []string{"disabled"},
			},
		},
	}

	testNamespaceSelector := &metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{
				Key:      "kubernetes.io/metadata.name",
				Operator: metav1.LabelSelectorOpIn,
				Values:   []string{"falcon-clusterguard-test"},
			},
		},
	}

	return &arv1.ValidatingWebhookConfiguration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: arv1.SchemeGroupVersion.String(),
			Kind:       "ValidatingWebhookConfiguration",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: webhookName,
			Labels: map[string]string{
				"app":                            a.prefix(),
				pkgcommon.KubernetesNameKey:      a.prefix(),
				pkgcommon.KubernetesComponentKey: pkgcommon.AdmissionComponentName,
				pkgcommon.FalconProviderKey:      pkgcommon.FalconProviderValue,
			},
		},
		Webhooks: []arv1.ValidatingWebhook{
			{
				Name:                    webhookName,
				AdmissionReviewVersions: []string{"v1"},
				SideEffects:             &sideEffects,
				FailurePolicy:           &failurePolicy,
				MatchPolicy:             &matchPolicy,
				TimeoutSeconds:          &timeoutSeconds,
				ClientConfig: arv1.WebhookClientConfig{
					CABundle: caBundle,
					Service: &arv1.ServiceReference{
						Name:      pkgcommon.AdmissionWebhookServiceName,
						Namespace: namespace,
						Path:      &path,
						Port:      &port,
					},
				},
				NamespaceSelector: namespaceSelector,
				Rules: []arv1.RuleWithOperations{
					{
						Operations: []arv1.OperationType{arv1.Create, arv1.Update},
						Rule: arv1.Rule{
							APIGroups:   []string{""},
							APIVersions: []string{"v1"},
							Resources:   []string{"pods", "pods/ephemeralcontainers"},
							Scope:       &scope,
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
				TimeoutSeconds:          &timeoutSeconds,
				ClientConfig: arv1.WebhookClientConfig{
					CABundle: caBundle,
					Service: &arv1.ServiceReference{
						Name:      pkgcommon.AdmissionWebhookServiceName,
						Namespace: namespace,
						Path:      &path,
						Port:      &port,
					},
				},
				NamespaceSelector: namespaceSelector,
				Rules: []arv1.RuleWithOperations{
					{
						Operations: []arv1.OperationType{arv1.Create, arv1.Update},
						Rule: arv1.Rule{
							APIGroups:   []string{""},
							APIVersions: []string{"v1"},
							Resources:   []string{"replicationcontrollers", "services"},
							Scope:       &scope,
						},
					},
					{
						Operations: []arv1.OperationType{arv1.Create, arv1.Update},
						Rule: arv1.Rule{
							APIGroups:   []string{"apps"},
							APIVersions: []string{"v1"},
							Resources:   []string{"daemonsets", "deployments", "replicasets", "statefulsets"},
							Scope:       &scope,
						},
					},
					{
						Operations: []arv1.OperationType{arv1.Create, arv1.Update},
						Rule: arv1.Rule{
							APIGroups:   []string{"batch"},
							APIVersions: []string{"v1"},
							Resources:   []string{"cronjobs", "jobs"},
							Scope:       &scope,
						},
					},
				},
			},
			{
				Name:                    "test." + webhookName,
				AdmissionReviewVersions: []string{"v1"},
				SideEffects:             &sideEffects,
				FailurePolicy:           &failurePolicy,
				MatchPolicy:             &matchPolicy,
				TimeoutSeconds:          &timeoutSeconds,
				ClientConfig: arv1.WebhookClientConfig{
					CABundle: caBundle,
					Service: &arv1.ServiceReference{
						Name:      pkgcommon.AdmissionWebhookServiceName,
						Namespace: namespace,
						Path:      &path,
						Port:      &port,
					},
				},
				NamespaceSelector: testNamespaceSelector,
				Rules: []arv1.RuleWithOperations{
					{
						Operations: []arv1.OperationType{arv1.Delete},
						Rule: arv1.Rule{
							APIGroups:   []string{""},
							APIVersions: []string{"v1"},
							Resources:   []string{"pods", "pods/ephemeralcontainers"},
							Scope:       &scope,
						},
					},
					{
						Operations: []arv1.OperationType{arv1.Delete},
						Rule: arv1.Rule{
							APIGroups:   []string{""},
							APIVersions: []string{"v1"},
							Resources:   []string{"replicationcontrollers", "services"},
							Scope:       &scope,
						},
					},
					{
						Operations: []arv1.OperationType{arv1.Delete},
						Rule: arv1.Rule{
							APIGroups:   []string{"apps"},
							APIVersions: []string{"v1"},
							Resources:   []string{"daemonsets", "deployments", "replicasets", "statefulsets"},
							Scope:       &scope,
						},
					},
					{
						Operations: []arv1.OperationType{arv1.Delete},
						Rule: arv1.Rule{
							APIGroups:   []string{"batch"},
							APIVersions: []string{"v1"},
							Resources:   []string{"cronjobs", "jobs"},
							Scope:       &scope,
						},
					},
				},
			},
		},
	}
}

// namespaceSelectorNeedsUpdate returns true if any expression present in desired is
// absent from or differs in existing. Keys not present in desired are considered
// platform-managed and are ignored.
func namespaceSelectorNeedsUpdate(desired, existing *metav1.LabelSelector) bool {
	if desired == nil {
		return existing != nil
	}
	if existing == nil {
		return true
	}
	existingByKey := map[string]metav1.LabelSelectorRequirement{}
	for _, expr := range existing.MatchExpressions {
		existingByKey[expr.Key] = expr
	}
	for _, desiredExpr := range desired.MatchExpressions {
		existingExpr, ok := existingByKey[desiredExpr.Key]
		if !ok || !reflect.DeepEqual(desiredExpr, existingExpr) {
			return true
		}
	}
	return false
}

// mergeNamespaceSelector returns a selector containing all expressions from desired
// plus any expressions in existing whose keys are not present in desired.
func mergeNamespaceSelector(desired, existing *metav1.LabelSelector) *metav1.LabelSelector {
	if existing == nil {
		return desired
	}
	desiredKeys := make(map[string]struct{}, len(desired.MatchExpressions))
	for _, expr := range desired.MatchExpressions {
		desiredKeys[expr.Key] = struct{}{}
	}
	merged := make([]metav1.LabelSelectorRequirement, 0, len(desired.MatchExpressions))
	for _, expr := range existing.MatchExpressions {
		if _, inDesired := desiredKeys[expr.Key]; !inDesired {
			merged = append(merged, expr)
		}
	}
	merged = append(merged, desired.MatchExpressions...)
	return &metav1.LabelSelector{MatchExpressions: merged}
}

// reconcileValidatingWebhook returns true if the webhook configuration was updated, which requires a pod restart.
func (a *ClusterGuardController) reconcileValidatingWebhook(ctx context.Context, caBundle []byte) (bool, error) {
	webhook := a.ValidatingWebhook(caBundle)
	existing := &arv1.ValidatingWebhookConfiguration{}
	found, err := k8sutils.GetOrCreate(ctx, a.r, a.cfg.Request, a.cfg.Owner, a.cfg.Status, webhook, existing,
		types.NamespacedName{Name: pkgcommon.AdmissionValidatingWebhookName},
		"Failed to get FalconClusterGuard ValidatingWebhookConfiguration")
	if !found || err != nil {
		return false, err
	}

	needsUpdate := len(webhook.Webhooks) != len(existing.Webhooks)
	if !needsUpdate && len(webhook.Webhooks) > 0 {
		if !reflect.DeepEqual(webhook.Webhooks[0].FailurePolicy, existing.Webhooks[0].FailurePolicy) {
			a.r.GetLog().V(1).Info("Updating FalconClusterGuard ValidatingWebhookConfiguration: FailurePolicy changed",
				"old", existing.Webhooks[0].FailurePolicy,
				"new", webhook.Webhooks[0].FailurePolicy)
			needsUpdate = true
		}
		if !reflect.DeepEqual(webhook.Webhooks[0].ClientConfig, existing.Webhooks[0].ClientConfig) {
			a.r.GetLog().V(1).Info("Updating FalconClusterGuard ValidatingWebhookConfiguration: ClientConfig changed")
			needsUpdate = true
		}
		if namespaceSelectorNeedsUpdate(webhook.Webhooks[0].NamespaceSelector, existing.Webhooks[0].NamespaceSelector) {
			a.r.GetLog().V(1).Info("Updating FalconClusterGuard ValidatingWebhookConfiguration: NamespaceSelector changed")
			needsUpdate = true
		}
	}

	if needsUpdate {
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			if err := pkgcommon.GetNamespacedObject(ctx, a.r, a.r.GetK8sReader(),
				types.NamespacedName{Name: pkgcommon.AdmissionValidatingWebhookName},
				existing); err != nil {
				return err
			}
			for i := range webhook.Webhooks {
				if i < len(existing.Webhooks) {
					webhook.Webhooks[i].NamespaceSelector = mergeNamespaceSelector(
						webhook.Webhooks[i].NamespaceSelector,
						existing.Webhooks[i].NamespaceSelector,
					)
				}
			}
			existing.Webhooks = webhook.Webhooks
			existing.SetGroupVersionKind(arv1.SchemeGroupVersion.WithKind("ValidatingWebhookConfiguration"))
			return k8sutils.Update(a.r, ctx, a.cfg.Request, a.r.GetLog(), a.cfg.Owner, a.cfg.Status, existing)
		})
		return err == nil, err
	}
	return false, nil
}
