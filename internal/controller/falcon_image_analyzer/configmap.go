package falcon

import (
	"context"
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

const (
	isKubernetes            = "true"
	enableDebug             = "false"
	agentRunmode            = "watcher"
	agentMaxConsumerThreads = "1"
)

func (r *FalconImageAnalyzerReconciler) reconcileConfigMap(ctx context.Context, req ctrl.Request, log logr.Logger, falconImageAnalyzer *falconv1alpha1.FalconImageAnalyzer) (bool, error) {
	log.Info("config map")
	return r.reconcileGenericConfigMap(falconImageAnalyzer.Name+"-config", r.newConfigMap, ctx, req, log, falconImageAnalyzer)
}

func (r *FalconImageAnalyzerReconciler) reconcileGenericConfigMap(name string, genFunc func(context.Context, string, *falconv1alpha1.FalconImageAnalyzer) (*corev1.ConfigMap, error), ctx context.Context, req ctrl.Request, log logr.Logger, falconImageAnalyzer *falconv1alpha1.FalconImageAnalyzer) (bool, error) {
	cm, err := genFunc(ctx, name, falconImageAnalyzer)
	if err != nil {
		return false, err
	}

	existingCM := &corev1.ConfigMap{}
	err = r.Get(ctx, types.NamespacedName{Name: name, Namespace: falconImageAnalyzer.Spec.InstallNamespace}, existingCM)
	if err != nil && apierrors.IsNotFound(err) {
		err = k8sutils.Create(r.Client, r.Scheme, ctx, req, log, falconImageAnalyzer, &falconImageAnalyzer.Status, cm)
		if err != nil {
			return false, err
		}

		return false, nil
	} else if err != nil {
		log.Error(err, "Failed to get FalconImageAnalyzer ConfigMap")
		return false, err
	}

	if !reflect.DeepEqual(cm.Data, existingCM.Data) {
		existingCM.Data = cm.Data
		if err := k8sutils.Update(r.Client, ctx, req, log, falconImageAnalyzer, &falconImageAnalyzer.Status, existingCM); err != nil {
			return false, err
		}
		return true, nil
	}

	return false, nil

}

func (r *FalconImageAnalyzerReconciler) newConfigMap(ctx context.Context, name string, falconImageAnalyzer *falconv1alpha1.FalconImageAnalyzer) (*corev1.ConfigMap, error) {
	var err error
	data := common.MakeSensorEnvMap(falconImageAnalyzer.Spec.Falcon)

	cid := ""
	if falconImageAnalyzer.Spec.Falcon.CID != nil {
		cid = *falconImageAnalyzer.Spec.Falcon.CID
	}

	if cid == "" && falconImageAnalyzer.Spec.FalconAPI != nil {
		cid, err = falcon_api.FalconCID(ctx, falconImageAnalyzer.Spec.FalconAPI.CID, falconImageAnalyzer.Spec.FalconAPI.ApiConfig())
		if err != nil {
			return &corev1.ConfigMap{}, err
		}
	}

	if falconImageAnalyzer.Spec.FalconAPI.ClientId != "" {
		data["AGENT_CLIENT_ID"] = falconImageAnalyzer.Spec.FalconAPI.ClientId
	}

	if falconImageAnalyzer.Spec.FalconAPI.ClientId != "" {
		data["AGENT_CLIENT_SECRET"] = falconImageAnalyzer.Spec.FalconAPI.ClientSecret
	}

	if falconImageAnalyzer.Spec.FalconAPI.CloudRegion != "" {
		data["AGENT_REGION"] = falconImageAnalyzer.Spec.FalconAPI.CloudRegion
	}

	data["IS_KUBERNETES"] = isKubernetes
	data["AGENT_CID"] = cid
	data["AGENT_DEBUG"] = enableDebug
	data["AGENT_RUNMODE"] = agentRunmode
	data["AGENT_MAX_CONSUMER_THREADS"] = agentMaxConsumerThreads

	/*
		AGENT_CLUSTER_NAME: {{ .Values.crowdstrikeConfig.clusterName | quote }}
		AGENT_REGISTRY_CREDENTIALS: {{ .Values.privateRegistries.credentials | quote }}
		AGENT_NAMESPACE_EXCLUSIONS: {{ .Values.exclusions.namespace | quote }}
		AGENT_REGISTRY_EXCLUSIONS: {{ .Values.exclusions.registry | quote }}
		AGENT_REGION: {{ .Values.crowdstrikeConfig.agentRegion | quote }}
		AGENT_TEMP_MOUNT_SIZE: {{ include "falcon-image-analyzer.tempvolsize" . | quote }}
	*/
	return assets.SensorConfigMap(name, falconImageAnalyzer.Spec.InstallNamespace, common.FalconImageAnalyzer, data), nil
}
