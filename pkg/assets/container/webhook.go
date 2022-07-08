package container

import (
	"fmt"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	arv1 "k8s.io/api/admissionregistration/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func ContainerMutatingWebhook(name string, namespace string, falconContainer *falconv1alpha1.FalconContainer) *arv1.MutatingWebhookConfiguration {
	sideEffectNone := arv1.SideEffectClassNone
	failurePolicy := arv1.Fail
	url := falconContainer.Spec.FalconContainerSensorConfig.URL
	timeoutSeconds := falconContainer.Spec.FalconContainerSensorConfig.TimeoutSeconds
	disableNSInjection := falconContainer.Spec.FalconContainerSensorConfig.DisableNSInjection
	webhookName := fmt.Sprintf("%s.%s.svc", name, namespace)

	return &arv1.MutatingWebhookConfiguration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				common.FalconInstanceNameKey: name,
				common.FalconInstanceKey:     "container_sensor",
				common.FalconComponentKey:    "container_sensor",
				common.FalconManagedByKey:    name,
				common.FalconProviderKey:     "CrowdStrike",
				common.FalconPartOfKey:       "Falcon",
				common.FalconControllerKey:   "controller-manager",
			},
		},
		Webhooks: []arv1.MutatingWebhook{
			{
				Name:                    webhookName,
				AdmissionReviewVersions: common.FCAdmissionReviewVersions(),
				SideEffects:             &sideEffectNone,
				FailurePolicy:           &failurePolicy,
				ClientConfig:            webhookClientConfig(name, namespace, url),
				TimeoutSeconds:          &timeoutSeconds,
				NamespaceSelector:       webhookMatchExpressions(disableNSInjection, namespace),
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

func webhookClientConfig(name string, namespace string, url string) arv1.WebhookClientConfig {
	path := "/mutate"

	webhookConfig := arv1.WebhookClientConfig{
		CABundle: []byte(common.CertAuth.Cert),
	}

	if len(url) > 0 {
		url := fmt.Sprintf("https://%s:%d/mutate", url, common.FalconServiceHTTPSPort)
		webhookConfig.URL = &url
	} else {
		webhookConfig.Service = &arv1.ServiceReference{
			Namespace: namespace,
			Name:      name,
			Path:      &path,
		}
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
