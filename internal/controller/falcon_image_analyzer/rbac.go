package falcon

import (
	"context"
	"reflect"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/internal/controller/assets"
	k8sutils "github.com/crowdstrike/falcon-operator/internal/controller/common"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	types "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	imageClusterRoleName        = "falcon-operator-image-controller-role"
	imageClusterRoleBindingName = "falcon-operator-image-controller-rolebinding"
)

// reconcileServiceAccount only returns true when a service account update requires a deployment restart.
// Only updates to image pull secrets should trigger a deployment restart.
func (r *FalconImageAnalyzerReconciler) reconcileServiceAccount(ctx context.Context, req ctrl.Request, log logr.Logger, falconImageAnalyzer *falconv1alpha1.FalconImageAnalyzer) (bool, error) {
	serviceAccountUpdated := false
	imagePullSecretsUpdated := false
	existingServiceAccount := &corev1.ServiceAccount{}

	imagePullSecrets := []corev1.LocalObjectReference{{Name: common.FalconPullSecretName}}
	for _, secret := range falconImageAnalyzer.Spec.ImageAnalyzerConfig.ImagePullSecrets {
		if secret.Name != common.FalconPullSecretName {
			imagePullSecrets = append(imagePullSecrets, corev1.LocalObjectReference{Name: secret.Name})
		}
	}

	serviceAccount := assets.ServiceAccount(common.ImageServiceAccountName,
		falconImageAnalyzer.Spec.InstallNamespace,
		common.FalconImageAnalyzer,
		falconImageAnalyzer.Spec.ImageAnalyzerConfig.ServiceAccount.Annotations,
		imagePullSecrets)

	err := common.GetNamespacedObject(ctx, r.Client, r.Reader, types.NamespacedName{Name: common.ImageServiceAccountName, Namespace: falconImageAnalyzer.Spec.InstallNamespace}, existingServiceAccount)
	if err != nil && apierrors.IsNotFound(err) {
		err = k8sutils.Create(r.Client, r.Scheme, ctx, req, log, falconImageAnalyzer, &falconImageAnalyzer.Status, serviceAccount)
		if err != nil {
			return serviceAccountUpdated, err
		}

		return serviceAccountUpdated, nil
	} else if err != nil {
		log.Error(err, "Failed to get FalconImageAnalyzer ServiceAccount")
		return serviceAccountUpdated, err
	}

	// Set GVK on existingServiceAccount since it's not populated when retrieved from the API server
	existingServiceAccount.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ServiceAccount"))

	// Check if any annotations from serviceAccount need to be added to existingServiceAccount
	if serviceAccount.ObjectMeta.Annotations != nil {
		if existingServiceAccount.ObjectMeta.Annotations == nil {
			existingServiceAccount.ObjectMeta.Annotations = make(map[string]string)
		}
		for key, value := range serviceAccount.ObjectMeta.Annotations {
			if existingValue, exists := existingServiceAccount.ObjectMeta.Annotations[key]; !exists || existingValue != value {
				existingServiceAccount.ObjectMeta.Annotations[key] = value
				serviceAccountUpdated = true
			}
		}
	}

	// Check if any labels from serviceAccount need to be added to existingServiceAccount
	if serviceAccount.ObjectMeta.Labels != nil {
		if existingServiceAccount.ObjectMeta.Labels == nil {
			existingServiceAccount.ObjectMeta.Labels = make(map[string]string)
		}
		for key, value := range serviceAccount.ObjectMeta.Labels {
			if existingValue, exists := existingServiceAccount.ObjectMeta.Labels[key]; !exists || existingValue != value {
				existingServiceAccount.ObjectMeta.Labels[key] = value
				serviceAccountUpdated = true
			}
		}
	}

	if !reflect.DeepEqual(serviceAccount.ImagePullSecrets, existingServiceAccount.ImagePullSecrets) {
		existingServiceAccount.ImagePullSecrets = serviceAccount.ImagePullSecrets
		imagePullSecretsUpdated = true
	}

	if serviceAccountUpdated || imagePullSecretsUpdated {
		err = k8sutils.Update(r.Client, ctx, req, log, falconImageAnalyzer, &falconImageAnalyzer.Status, existingServiceAccount)
		if err != nil {
			return serviceAccountUpdated, err
		}
	}

	return imagePullSecretsUpdated, nil
}

func (r *FalconImageAnalyzerReconciler) reconcileClusterRoleBinding(ctx context.Context, req ctrl.Request, log logr.Logger, falconImageAnalyzer *falconv1alpha1.FalconImageAnalyzer) error {
	clusterRoleBinding := assets.ClusterRoleBinding(imageClusterRoleBindingName,
		falconImageAnalyzer.Spec.InstallNamespace,
		imageClusterRoleName,
		common.ImageServiceAccountName,
		common.FalconImageAnalyzer,
		[]rbacv1.Subject{})
	existingClusterRoleBinding := &rbacv1.ClusterRoleBinding{}

	err := common.GetNamespacedObject(ctx, r.Client, r.Reader, types.NamespacedName{Name: imageClusterRoleBindingName}, existingClusterRoleBinding)
	if err != nil && apierrors.IsNotFound(err) {
		err = k8sutils.Create(r.Client, r.Scheme, ctx, req, log, falconImageAnalyzer, &falconImageAnalyzer.Status, clusterRoleBinding)
		if err != nil {
			return err
		}

		return nil
	} else if err != nil {
		log.Error(err, "Failed to get FalconImageAnalyzer ClusterRoleBinding")
		return err
	}

	// If the RoleRef changes, we need to re-create it
	if !reflect.DeepEqual(clusterRoleBinding.RoleRef, existingClusterRoleBinding.RoleRef) {
		if err = k8sutils.Delete(r.Client, ctx, req, log, falconImageAnalyzer, &falconImageAnalyzer.Status, existingClusterRoleBinding); err != nil {
			return err
		}

		err = k8sutils.Create(r.Client, r.Scheme, ctx, req, log, falconImageAnalyzer, &falconImageAnalyzer.Status, clusterRoleBinding)
		if err != nil {
			return err
		}
		// If RoleRef is the same but Subjects have changed, update the object and post to k8s api
	} else if !reflect.DeepEqual(clusterRoleBinding.Subjects, existingClusterRoleBinding.Subjects) {
		existingClusterRoleBinding.Subjects = clusterRoleBinding.Subjects
		err = k8sutils.Update(r.Client, ctx, req, log, falconImageAnalyzer, &falconImageAnalyzer.Status, existingClusterRoleBinding)
		if err != nil {
			return err
		}
	}

	return nil
}
