package controllers

import (
	"context"
	"fmt"
	"reflect"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/internal/controller/assets"
	k8sutils "github.com/crowdstrike/falcon-operator/internal/controller/common"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/crowdstrike/falcon-operator/pkg/falcon_api"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	types "k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

func (r *FalconAdmissionReconciler) reconcileRegistryCABundleConfigMap(ctx context.Context, req ctrl.Request, log logr.Logger, falconAdmission *falconv1alpha1.FalconAdmission) (bool, error) {
	return r.reconcileGenericConfigMap(falconAdmission.Name+"-registry-certs", r.newCABundleConfigMap, ctx, req, log, falconAdmission)
}

func (r *FalconAdmissionReconciler) reconcileConfigMap(ctx context.Context, req ctrl.Request, log logr.Logger, falconAdmission *falconv1alpha1.FalconAdmission) (bool, error) {
	return r.reconcileGenericConfigMap(falconAdmission.Name+"-config", r.newConfigMap, ctx, req, log, falconAdmission)
}

func (r *FalconAdmissionReconciler) reconcileGenericConfigMap(name string, genFunc func(context.Context, string, *falconv1alpha1.FalconAdmission) (*corev1.ConfigMap, error), ctx context.Context, req ctrl.Request, log logr.Logger, falconAdmission *falconv1alpha1.FalconAdmission) (bool, error) {
	cm, err := genFunc(ctx, name, falconAdmission)
	if err != nil {
		return false, err
	}

	existingCM := &corev1.ConfigMap{}
	err = common.GetNamespacedObject(ctx, r.Client, r.Reader, types.NamespacedName{Name: name, Namespace: falconAdmission.Spec.InstallNamespace}, existingCM)

	if err != nil && apierrors.IsNotFound(err) {
		err = k8sutils.Create(r.Client, r.Scheme, ctx, req, log, falconAdmission, &falconAdmission.Status, cm)
		if err != nil {
			return false, err
		}

		return false, nil
	} else if err != nil {
		log.Error(err, "Failed to get FalconAdmission ConfigMap")
		return false, err
	}

	if !reflect.DeepEqual(cm.Data, existingCM.Data) {
		existingCM.Data = cm.Data
		if err := k8sutils.Update(r.Client, ctx, req, log, falconAdmission, &falconAdmission.Status, existingCM); err != nil {
			return false, err
		}
		return true, nil
	}

	return false, nil

}

func (r *FalconAdmissionReconciler) newCABundleConfigMap(ctx context.Context, name string, falconAdmission *falconv1alpha1.FalconAdmission) (*corev1.ConfigMap, error) {
	data := make(map[string]string)
	if falconAdmission.Spec.Registry.TLS.CACertificate != "" {
		data["tls.crt"] = string(common.DecodeBase64Interface(falconAdmission.Spec.Registry.TLS.CACertificate))

		return assets.SensorConfigMap(name, falconAdmission.Spec.InstallNamespace, common.FalconSidecarSensor, data), nil
	}
	return &corev1.ConfigMap{}, fmt.Errorf("unable to determine contents of Registry TLS CACertificate attribute")
}

func (r *FalconAdmissionReconciler) newConfigMap(ctx context.Context, name string, falconAdmission *falconv1alpha1.FalconAdmission) (*corev1.ConfigMap, error) {
	var err error
	data := common.MakeSensorEnvMap(falconAdmission.Spec.Falcon)

	cid := ""
	if falconAdmission.Spec.Falcon.CID != nil {
		cid = *falconAdmission.Spec.Falcon.CID
	}

	if cid == "" && falconAdmission.Spec.FalconAPI != nil {
		cid, err = falcon_api.FalconCID(ctx, falconAdmission.Spec.FalconAPI.CID, falconAdmission.Spec.FalconAPI.ApiConfig())
		if err != nil {
			return &corev1.ConfigMap{}, err
		}
	}
	data["FALCONCTL_OPT_CID"] = cid

	return assets.SensorConfigMap(name, falconAdmission.Spec.InstallNamespace, common.FalconAdmissionController, data), nil
}
