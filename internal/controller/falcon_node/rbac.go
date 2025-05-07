package falcon

import (
	"context"
	"reflect"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/internal/controller/assets"
	"github.com/crowdstrike/falcon-operator/pkg/common"

	"github.com/go-logr/logr"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

// #nosec G101
const (
	nodeSensorSecretReaderRoleName        = "falcon-node-sensor-secret-reader-role"
	nodeSensorSecretReaderRoleBindingName = "falcon-node-sensor-secret-reader-role-binding"
)

func (r *FalconNodeSensorReconciler) reconcileFalconSecretRole(
	ctx context.Context,
	log logr.Logger,
	nodeSensor *falconv1alpha1.FalconNodeSensor,
) error {
	falconSecretReaderRole := assets.FalconSecretReaderRole(
		nodeSensorSecretReaderRoleName,
		nodeSensor.Spec.FalconSecret.Namespace,
		common.FalconKernelSensor,
	)
	existingRole := &rbacv1.Role{}

	err := common.GetNamespacedObject(ctx, r.Client, r.Reader, types.NamespacedName{Name: nodeSensorSecretReaderRoleName, Namespace: nodeSensor.Spec.FalconSecret.Namespace}, existingRole)
	if err != nil && apierrors.IsNotFound(err) {
		err = r.Create(ctx, falconSecretReaderRole)
		if err != nil {
			return err
		}

		return nil
	} else if err != nil {
		log.Error(err, "Failed to get FalconNodeSensor secret reader Role")
		return err
	}

	if !reflect.DeepEqual(falconSecretReaderRole.Rules, existingRole.Rules) {
		existingRole.Rules = falconSecretReaderRole.Rules
		err = r.Update(ctx, existingRole)
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *FalconNodeSensorReconciler) reconcileFalconSecretRoleBinding(
	ctx context.Context,
	log logr.Logger,
	nodeSensor *falconv1alpha1.FalconNodeSensor,
) error {
	secretReaderRoleBinding := assets.FalconSecretReaderRoleBinding(
		nodeSensorSecretReaderRoleBindingName,
		nodeSensor.Spec.InstallNamespace,
		nodeSensorSecretReaderRoleName,
		common.NodeServiceAccountName,
		common.FalconKernelSensor,
	)
	existingRoleBinding := &rbacv1.RoleBinding{}

	err := r.Get(ctx, types.NamespacedName{Name: nodeSensorSecretReaderRoleBindingName, Namespace: nodeSensor.Spec.InstallNamespace}, existingRoleBinding)
	if err != nil && apierrors.IsNotFound(err) {
		err = r.Create(ctx, secretReaderRoleBinding)
		if err != nil {
			return err
		}

		return nil
	} else if err != nil {
		log.Error(err, "Failed to get FalconNodeSensor secret reader RoleBinding")
		return err
	}

	// If the RoleRef changes, we need to re-create it
	if !reflect.DeepEqual(secretReaderRoleBinding.RoleRef, existingRoleBinding.RoleRef) {
		if err = r.Delete(ctx, existingRoleBinding); err != nil {
			return err
		}

		err = r.Create(ctx, secretReaderRoleBinding)
		if err != nil {
			return err
		}
		// If RoleRef is the same but Subjects have changed, update the object and post to k8s api
	} else if !reflect.DeepEqual(secretReaderRoleBinding.Subjects, existingRoleBinding.Subjects) {
		existingRoleBinding.Subjects = secretReaderRoleBinding.Subjects
		err = r.Update(ctx, existingRoleBinding)
		if err != nil {
			return err
		}
	}

	return nil
}
