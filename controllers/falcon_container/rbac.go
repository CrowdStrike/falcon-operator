package falcon

import (
	"context"
	"fmt"
	"reflect"

	"github.com/crowdstrike/falcon-operator/apis/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	injectorClusterRoleName        = "falcon-operator-container-role"
	injectorClusterRoleBindingName = "read-secrets-global"
)

func (r *FalconContainerReconciler) reconcileServiceAccount(ctx context.Context, log logr.Logger, falconContainer *v1alpha1.FalconContainer) (*corev1.ServiceAccount, error) {
	update := false
	serviceAccount := r.newServiceAccount(falconContainer)
	existingServiceAccount := &corev1.ServiceAccount{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: common.SidecarServiceAccountName, Namespace: r.Namespace()}, existingServiceAccount)
	if err != nil {
		if errors.IsNotFound(err) {
			if err = ctrl.SetControllerReference(falconContainer, serviceAccount, r.Scheme); err != nil {
				// Set the service account controller reference, but only if we create it; not on updates to an existing sa
				return &corev1.ServiceAccount{}, fmt.Errorf("unable to set controller reference on service account %s: %v", serviceAccount.ObjectMeta.Name, err)
			}
			return serviceAccount, r.Create(ctx, log, falconContainer, serviceAccount)
		}
		return &corev1.ServiceAccount{}, fmt.Errorf("unable to query existing service account %s: %v", common.SidecarServiceAccountName, err)
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
		return existingServiceAccount, r.Update(ctx, log, falconContainer, existingServiceAccount)
	}
	return existingServiceAccount, nil

}

func (r *FalconContainerReconciler) reconcileClusterRoleBinding(ctx context.Context, log logr.Logger, falconContainer *v1alpha1.FalconContainer) (*rbacv1.ClusterRoleBinding, error) {
	clusterRoleBinding := r.newClusterRoleBinding(falconContainer)
	existingClusterRoleBinding := &rbacv1.ClusterRoleBinding{}
	err := r.Client.Get(ctx, types.NamespacedName{Name: injectorClusterRoleBindingName}, existingClusterRoleBinding)
	if err != nil {
		if errors.IsNotFound(err) {
			if err = ctrl.SetControllerReference(falconContainer, clusterRoleBinding, r.Scheme); err != nil {
				return &rbacv1.ClusterRoleBinding{}, fmt.Errorf("unable to set controller reference on cluster role binding %s: %v", clusterRoleBinding.ObjectMeta.Name, err)
			}
			return clusterRoleBinding, r.Create(ctx, log, falconContainer, clusterRoleBinding)
		}
		return &rbacv1.ClusterRoleBinding{}, fmt.Errorf("unable to query existing cluster role binding %s: %v", injectorClusterRoleBindingName, err)
	}
	// If the RoleRef changes, we need to re-create it
	if !reflect.DeepEqual(clusterRoleBinding.RoleRef, existingClusterRoleBinding.RoleRef) {
		if err = ctrl.SetControllerReference(falconContainer, clusterRoleBinding, r.Scheme); err != nil {
			return &rbacv1.ClusterRoleBinding{}, fmt.Errorf("unable to set controller reference on cluster role binding %s: %v", clusterRoleBinding.ObjectMeta.Name, err)
		}
		if err = r.Delete(ctx, log, falconContainer, existingClusterRoleBinding); err != nil {
			return &rbacv1.ClusterRoleBinding{}, fmt.Errorf("unable to delete existing cluster role binding %s: %v", injectorClusterRoleBindingName, err)
		}
		return clusterRoleBinding, r.Create(ctx, log, falconContainer, clusterRoleBinding)
		// If RoleRef is the same but Subjects have changed, update the object and post to k8s api
	} else if !reflect.DeepEqual(clusterRoleBinding.Subjects, existingClusterRoleBinding.Subjects) {
		existingClusterRoleBinding.Subjects = clusterRoleBinding.Subjects
		return existingClusterRoleBinding, r.Update(ctx, log, falconContainer, existingClusterRoleBinding)
	}
	return existingClusterRoleBinding, nil

}

func (r *FalconContainerReconciler) newServiceAccount(falconContainer *v1alpha1.FalconContainer) *corev1.ServiceAccount {
	imagePullSecrets := []corev1.LocalObjectReference{{Name: common.FalconPullSecretName}}
	if common.FalconPullSecretName != falconContainer.Spec.Injector.ImagePullSecretName {
		imagePullSecrets = append(imagePullSecrets, corev1.LocalObjectReference{Name: falconContainer.Spec.Injector.ImagePullSecretName})
	}
	return &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        common.SidecarServiceAccountName,
			Namespace:   r.Namespace(),
			Labels:      FcLabels,
			Annotations: falconContainer.Spec.Injector.ServiceAccount.Annotations,
		},
		ImagePullSecrets: imagePullSecrets,
	}
}

func (r *FalconContainerReconciler) newClusterRoleBinding(falconContainer *v1alpha1.FalconContainer) *rbacv1.ClusterRoleBinding {
	return &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   injectorClusterRoleBindingName,
			Labels: FcLabels,
		},
		Subjects: []rbacv1.Subject{{
			Kind:      "ServiceAccount",
			Name:      common.SidecarServiceAccountName,
			Namespace: r.Namespace(),
		}},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     injectorClusterRoleName,
		},
	}
}
