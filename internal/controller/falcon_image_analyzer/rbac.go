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

func (r *FalconImageAnalyzerReconciler) reconcileServiceAccount(ctx context.Context, req ctrl.Request, log logr.Logger, falconImageAnalyzer *falconv1alpha1.FalconImageAnalyzer) error {
	update := false
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

	err := r.Get(ctx, types.NamespacedName{Name: common.ImageServiceAccountName, Namespace: falconImageAnalyzer.Spec.InstallNamespace}, existingServiceAccount)
	if err != nil && apierrors.IsNotFound(err) {
		err = k8sutils.Create(r.Client, r.Scheme, ctx, req, log, falconImageAnalyzer, &falconImageAnalyzer.Status, serviceAccount)
		if err != nil {
			return err
		}

		return nil
	} else if err != nil {
		log.Error(err, "Failed to get FalconImageAnalyzer ServiceAccount")
		return err
	}

	if !reflect.DeepEqual(serviceAccount.ObjectMeta.Annotations, existingServiceAccount.ObjectMeta.Annotations) {
		existingServiceAccount.ObjectMeta.Annotations = serviceAccount.ObjectMeta.Annotations
		update = true
	}
	if !reflect.DeepEqual(serviceAccount.ObjectMeta.Labels, existingServiceAccount.ObjectMeta.Labels) {
		existingServiceAccount.ObjectMeta.Labels = serviceAccount.ObjectMeta.Labels
		update = true
	}

	if update {
		err = k8sutils.Update(r.Client, ctx, req, log, falconImageAnalyzer, &falconImageAnalyzer.Status, existingServiceAccount)
		if err != nil {
			return err
		}
	}

	return nil

}

func (r *FalconImageAnalyzerReconciler) reconcileClusterRoleBinding(ctx context.Context, req ctrl.Request, log logr.Logger, falconImageAnalyzer *falconv1alpha1.FalconImageAnalyzer) error {
	clusterRoleBinding := assets.ClusterRoleBinding(imageClusterRoleBindingName,
		falconImageAnalyzer.Spec.InstallNamespace,
		imageClusterRoleName,
		common.ImageServiceAccountName,
		common.FalconImageAnalyzer,
		[]rbacv1.Subject{})
	existingClusterRoleBinding := &rbacv1.ClusterRoleBinding{}

	err := r.Client.Get(ctx, types.NamespacedName{Name: imageClusterRoleBindingName}, existingClusterRoleBinding)
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
