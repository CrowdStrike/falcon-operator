package container

import (
	"fmt"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	arv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ContainerMutatingWebhook(dsName string, nsName string, falconContainer *falconv1alpha1.FalconContainer) *arv1.MutatingWebhookConfiguration {
	return containerMutatingWebhook(dsName, nsName, falconContainer)
}

func containerMutatingWebhook(dsName string, nsName string, falconContainer *falconv1alpha1.FalconContainer) *arv1.MutatingWebhookConfiguration {
	sideEffectNone := arv1.SideEffectClassNone
	failurePolicy := arv1.Fail
	var timeoutSeconds int32 = 30
	useURL := false
	nsInjection := false
	fqdn := ""

	return &arv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      dsName,
			Namespace: nsName,
			Labels: map[string]string{
				common.FalconInstanceNameKey: dsName,
				common.FalconInstanceKey:     "container_sensor",
				common.FalconComponentKey:    "container_sensor",
				common.FalconManagedByKey:    dsName,
				common.FalconProviderKey:     common.FalconProviderValue,
				common.FalconPartOfKey:       "Falcon",
				common.FalconControllerKey:   "controller-manager",
			},
		},
		Webhooks: []arv1.MutatingWebhook{
			{
				Name:                    dsName + ".local.svc",
				AdmissionReviewVersions: common.FCAdmissionReviewVersions(),
				SideEffects:             &sideEffectNone,
				FailurePolicy:           &failurePolicy,
				ClientConfig:            webhookClientConfig(useURL, dsName, fqdn),
				TimeoutSeconds:          &timeoutSeconds,
				NamespaceSelector:       webhookMatchExpressions(nsInjection, dsName),
				Rules: []arv1.RuleWithOperations{
					{
						Operations: []arv1.OperationType{arv1.Create},
						Rule: arv1.Rule{
							APIGroups:   []string{""},
							APIVersions: []string{"v1"},
							Resources:   []string{"pods"},
						},
					},
				},
			},
		},
	}
}

func webhookClientConfig(useURL bool, namespace string, domain string) arv1.WebhookClientConfig {
	webhookConfig := arv1.WebhookClientConfig{
		CABundle: common.EncodedBase64String(common.CA.Cert),
	}
	path := "/mutate"
	url := fmt.Sprintf("https://%s:%d/mutate", domain, common.FalconServiceHTTPSPort)

	if !useURL {
		webhookConfig.Service = &arv1.ServiceReference{
			Namespace: namespace,
			Name:      namespace + "-injector",
			Path:      &path,
		}
	} else {
		webhookConfig.URL = &url
	}

	return webhookConfig
}

func webhookMatchExpressions(disableNSInjection bool, namespace string) *metav1.LabelSelector {
	operatorSelector := metav1.LabelSelectorOpNotIn
	operatorValue := []string{"disabled"}
	if disableNSInjection {
		operatorSelector = metav1.LabelSelectorOpIn
		operatorValue = []string{"enabled"}
	}

	return &metav1.LabelSelector{
		MatchExpressions: []metav1.LabelSelectorRequirement{
			{
				Key:      "sensor.crowdstrike.com/injection",
				Operator: operatorSelector,
				Values:   operatorValue,
			},
			{
				Key:      "sensor.falcon-system.crowdstrike.com/injection",
				Operator: operatorSelector,
				Values:   operatorValue,
			},
			{
				Key:      "name",
				Operator: metav1.LabelSelectorOpNotIn,
				Values:   []string{namespace},
			},
		},
	}
}
