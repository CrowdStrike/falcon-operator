package falcon

import (
	"context"
	"fmt"
	"reflect"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/internal/controller/assets"
	"github.com/go-logr/logr"
	arv1 "k8s.io/api/admissionregistration/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	types "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	webhookName = "mutatingwebhook.sidecar.falcon.crowdstrike.com"
)

func (r *FalconContainerReconciler) reconcileWebhook(ctx context.Context, log logr.Logger, falconContainer *falconv1alpha1.FalconContainer, caBundle []byte) (*arv1.MutatingWebhookConfiguration, error) {
	disableDefaultNSInjection := false

	if falconContainer.Spec.Injector.DisableDefaultNSInjection {
		disableDefaultNSInjection = falconContainer.Spec.Injector.DisableDefaultNSInjection
	}

	webhook := assets.MutatingWebhook(injectorName, r.Namespace(), webhookName, caBundle, disableDefaultNSInjection, falconContainer)
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
