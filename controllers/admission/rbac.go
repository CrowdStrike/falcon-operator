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
	admissionClusterRoleName        = "falcon-operator-admission-controller-role"
	admissionClusterRoleBindingName = "falcon-operator-admission-controller-rolebinding"
)

func (r *FalconAdmissionReconciler) reconcileServiceAccount(ctx context.Context, req ctrl.Request, log logr.Logger, falconAdmission *falconv1alpha1.FalconAdmission) error {
	update := false
	serviceAccount := assets.ServiceAccount(common.AdmissionServiceAccountName,
		falconAdmission.Spec.InstallNamespace,
		common.FalconAdmissionController,
		falconAdmission.Spec.AdmissionConfig.ServiceAccount.Annotations,
		falconAdmission.Spec.AdmissionConfig.ImagePullSecrets)
	existingServiceAccount := &corev1.ServiceAccount{}

	err := r.Get(ctx, types.NamespacedName{Name: common.AdmissionServiceAccountName, Namespace: falconAdmission.Spec.InstallNamespace}, existingServiceAccount)
	if err != nil && apierrors.IsNotFound(err) {
		err = k8sutils.Create(r.Client, r.Scheme, ctx, req, log, falconAdmission, &falconAdmission.Status, serviceAccount)
		if err != nil {
			return err
		}

		return nil
	} else if err != nil {
		log.Error(err, "Failed to get FalconAdmission ServiceAccount")
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
		err = k8sutils.Update(r.Client, ctx, req, log, falconAdmission, &falconAdmission.Status, existingServiceAccount)
		if err != nil {
			return err
		}
	}

	return nil

}

func (r *FalconAdmissionReconciler) reconcileClusterRoleBinding(ctx context.Context, req ctrl.Request, log logr.Logger, falconAdmission *falconv1alpha1.FalconAdmission) error {
	clusterRoleBinding := assets.ClusterRoleBinding(admissionClusterRoleBindingName,
		falconAdmission.Spec.InstallNamespace,
		admissionClusterRoleName,
		common.AdmissionServiceAccountName,
		common.FalconAdmissionController,
		[]rbacv1.Subject{})
	existingClusterRoleBinding := &rbacv1.ClusterRoleBinding{}

	err := r.Client.Get(ctx, types.NamespacedName{Name: admissionClusterRoleBindingName}, existingClusterRoleBinding)
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
	role := assets.Role("falcon-admission-controller-role", falconAdmission.Spec.InstallNamespace)
	existingRole := &rbacv1.Role{}

	err := r.Get(ctx, types.NamespacedName{Name: "falcon-admission-controller-role", Namespace: falconAdmission.Spec.InstallNamespace}, existingRole)
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
	roleBinding := assets.RoleBinding("falcon-admission-controller-role-binding",
		falconAdmission.Spec.InstallNamespace,
		"falcon-admission-controller-role",
		common.AdmissionServiceAccountName)
	existingRoleBinding := &rbacv1.RoleBinding{}

	err := r.Get(ctx, types.NamespacedName{Name: "falcon-admission-controller-role-binding", Namespace: falconAdmission.Spec.InstallNamespace}, existingRoleBinding)
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
