package controllers

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
	admissionClusterRoleName           = "falcon-operator-admission-controller-role"
	admissionClusterRoleBindingName    = "falcon-operator-admission-controller-rolebinding"
	admissionControllerRoleName        = "falcon-admission-controller-role"
	admissionControllerRoleBindingName = "falcon-admission-controller-role-binding"
)

func (r *FalconAdmissionReconciler) reconcileServiceAccount(ctx context.Context, req ctrl.Request, log logr.Logger, falconAdmission *falconv1alpha1.FalconAdmission) (bool, error) {
	serviceAccountUpdated := false
	imagePullSecretsUpdated := false
	existingServiceAccount := &corev1.ServiceAccount{}

	imagePullSecrets := []corev1.LocalObjectReference{{Name: common.FalconPullSecretName}}
	for _, secret := range falconAdmission.Spec.AdmissionConfig.ImagePullSecrets {
		if secret.Name != common.FalconPullSecretName {
			imagePullSecrets = append(imagePullSecrets, corev1.LocalObjectReference{Name: secret.Name})
		}
	}

	serviceAccount := assets.ServiceAccount(common.AdmissionServiceAccountName,
		falconAdmission.Spec.InstallNamespace,
		common.FalconAdmissionController,
		falconAdmission.Spec.AdmissionConfig.ServiceAccount.Annotations,
		imagePullSecrets)

	err := common.GetNamespacedObject(ctx, r.Client, r.Reader, types.NamespacedName{Name: common.AdmissionServiceAccountName, Namespace: falconAdmission.Spec.InstallNamespace}, existingServiceAccount)
	if err != nil && apierrors.IsNotFound(err) {
		err = k8sutils.Create(r.Client, r.Scheme, ctx, req, log, falconAdmission, &falconAdmission.Status, serviceAccount)
		if err != nil {
			return false, err
		}

		return serviceAccountUpdated, nil
	} else if err != nil {
		log.Error(err, "Failed to get FalconAdmission ServiceAccount")
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
		err = k8sutils.Update(r.Client, ctx, req, log, falconAdmission, &falconAdmission.Status, existingServiceAccount)
		if err != nil {
			return serviceAccountUpdated, err
		}
	}

	// Only trigger a pod restart if imagePullSecrets are updated
	return imagePullSecretsUpdated, nil
}

func (r *FalconAdmissionReconciler) reconcileClusterRoleBinding(ctx context.Context, req ctrl.Request, log logr.Logger, falconAdmission *falconv1alpha1.FalconAdmission) error {
	clusterRoleBinding := assets.ClusterRoleBinding(admissionClusterRoleBindingName,
		falconAdmission.Spec.InstallNamespace,
		admissionClusterRoleName,
		common.AdmissionServiceAccountName,
		common.FalconAdmissionController,
		[]rbacv1.Subject{})
	existingClusterRoleBinding := &rbacv1.ClusterRoleBinding{}

	err := common.GetNamespacedObject(ctx, r.Client, r.Reader, types.NamespacedName{Name: admissionClusterRoleBindingName}, existingClusterRoleBinding)
	if err != nil && apierrors.IsNotFound(err) {
		err = k8sutils.Create(r.Client, r.Scheme, ctx, req, log, falconAdmission, &falconAdmission.Status, clusterRoleBinding)
		if err != nil {
			return err
		}

		return nil
	} else if err != nil {
		log.Error(err, "Failed to get FalconAdmission ClusterRoleBinding")
		return err
	}

	// If the RoleRef changes, we need to re-create it
	if !reflect.DeepEqual(clusterRoleBinding.RoleRef, existingClusterRoleBinding.RoleRef) {
		if err = k8sutils.Delete(r.Client, ctx, req, log, falconAdmission, &falconAdmission.Status, existingClusterRoleBinding); err != nil {
			return err
		}

		err = k8sutils.Create(r.Client, r.Scheme, ctx, req, log, falconAdmission, &falconAdmission.Status, clusterRoleBinding)
		if err != nil {
			return err
		}
		// If RoleRef is the same but Subjects have changed, update the object and post to k8s api
	} else if !reflect.DeepEqual(clusterRoleBinding.Subjects, existingClusterRoleBinding.Subjects) {
		existingClusterRoleBinding.Subjects = clusterRoleBinding.Subjects
		err = k8sutils.Update(r.Client, ctx, req, log, falconAdmission, &falconAdmission.Status, existingClusterRoleBinding)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *FalconAdmissionReconciler) reconcileRole(ctx context.Context, req ctrl.Request, log logr.Logger, falconAdmission *falconv1alpha1.FalconAdmission) error {
	role := assets.Role(admissionControllerRoleName, falconAdmission.Spec.InstallNamespace)
	existingRole := &rbacv1.Role{}

	err := common.GetNamespacedObject(ctx, r.Client, r.Reader, types.NamespacedName{Name: admissionControllerRoleName, Namespace: falconAdmission.Spec.InstallNamespace}, existingRole)
	if err != nil && apierrors.IsNotFound(err) {
		err = k8sutils.Create(r.Client, r.Scheme, ctx, req, log, falconAdmission, &falconAdmission.Status, role)
		if err != nil {
			return err
		}

		return nil
	} else if err != nil {
		log.Error(err, "Failed to get FalconAdmission Role")
		return err
	}

	if !reflect.DeepEqual(role.Rules, existingRole.Rules) {
		existingRole.Rules = role.Rules
		err = k8sutils.Update(r.Client, ctx, req, log, falconAdmission, &falconAdmission.Status, existingRole)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *FalconAdmissionReconciler) reconcileRoleBinding(ctx context.Context, req ctrl.Request, log logr.Logger, falconAdmission *falconv1alpha1.FalconAdmission) error {
	roleBinding := assets.RoleBinding(admissionControllerRoleBindingName,
		falconAdmission.Spec.InstallNamespace,
		admissionControllerRoleName,
		common.AdmissionServiceAccountName)
	existingRoleBinding := &rbacv1.RoleBinding{}

	err := r.Get(ctx, types.NamespacedName{Name: admissionControllerRoleBindingName, Namespace: falconAdmission.Spec.InstallNamespace}, existingRoleBinding)
	if err != nil && apierrors.IsNotFound(err) {
		err = k8sutils.Create(r.Client, r.Scheme, ctx, req, log, falconAdmission, &falconAdmission.Status, roleBinding)
		if err != nil {
			return err
		}

		return nil
	} else if err != nil {
		log.Error(err, "Failed to get FalconAdmission RoleBinding")
		return err
	}

	// If the RoleRef changes, we need to re-create it
	if !reflect.DeepEqual(roleBinding.RoleRef, existingRoleBinding.RoleRef) {
		if err = k8sutils.Delete(r.Client, ctx, req, log, falconAdmission, &falconAdmission.Status, existingRoleBinding); err != nil {
			return err
		}

		err = k8sutils.Create(r.Client, r.Scheme, ctx, req, log, falconAdmission, &falconAdmission.Status, roleBinding)
		if err != nil {
			return err
		}
		// If RoleRef is the same but Subjects have changed, update the object and post to k8s api
	} else if !reflect.DeepEqual(roleBinding.Subjects, existingRoleBinding.Subjects) {
		existingRoleBinding.Subjects = roleBinding.Subjects
		err = k8sutils.Update(r.Client, ctx, req, log, falconAdmission, &falconAdmission.Status, existingRoleBinding)
		if err != nil {
			return err
		}
	}

	return nil
}
