package common

import (
	"context"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/crowdstrike/falcon-operator/pkg/falcon_secret"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

type FalconReconciler[T any] interface {
	GetK8sClient() client.Client
	GetK8sReader() client.Reader
}

func InjectFalconSecretData[T FalconReconciler[T], U FalconCRD](
	ctx context.Context,
	reconciler T,
	falconCrd U,
) error {
	secret := &corev1.Secret{}
	falconSecret := falconCrd.GetFalconSecretSpec()
	secretNamespacedName := types.NamespacedName{
		Name:      falconSecret.SecretName,
		Namespace: falconSecret.Namespace,
	}

	if err := common.GetNamespacedObject(ctx, reconciler.GetK8sClient(), reconciler.GetK8sReader(), secretNamespacedName, secret); err != nil {
		return err
	}

	// falconSpec does not apply to Falcon ImageAnalyzer
	falconSpec := falconCrd.GetFalconSpec()
	cid := falcon_secret.GetFalconCIDFromSecret(secret)
	provisioningToken := falcon_secret.GetFalconProvisioningTokenFromSecret(secret)

	falconSpec.CID = cid
	falconSpec.PToken = provisioningToken
	falconCrd.SetFalconSpec(falconSpec)

	falconApi := falconCrd.GetFalconAPISpec()
	if falconApi == nil {
		falconApi = &falconv1alpha1.FalconAPI{}
	}

	clientId, clientSecret := falcon_secret.GetFalconCredsFromSecret(secret)
	falconApi.ClientId = clientId
	falconApi.ClientSecret = clientSecret
	falconApi.CID = cid
	falconCrd.SetFalconAPISpec(falconApi)

	return nil
}
