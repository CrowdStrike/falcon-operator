package image_analyzer

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/crowdstrike/falcon-operator/internal/controller/assets"
	k8sutils "github.com/crowdstrike/falcon-operator/internal/controller/common"
	pkgcommon "github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/crowdstrike/gofalcon/falcon"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

const (
	isKubernetes            = "true"
	agentRunmode            = "watcher"
	agentMaxConsumerThreads = "1"
)

func (ia *ImageAnalyzer) reconcileConfigMap(ctx context.Context) (bool, error) {
	cm, err := ia.buildConfigMap()
	if err != nil {
		return false, err
	}

	existing := &corev1.ConfigMap{}
	err = pkgcommon.GetNamespacedObject(ctx, ia.r, ia.r.GetK8sReader(),
		types.NamespacedName{Name: pkgcommon.FCGImageAnalyzerConfigMapName, Namespace: ia.cfg.InstallNamespace}, existing)
	if err != nil && apierrors.IsNotFound(err) {
		return false, k8sutils.Create(ia.r, ia.r.GetScheme(), ctx, ia.cfg.Request, ia.r.GetLog(), ia.cfg.Owner, ia.cfg.Status, cm)
	} else if err != nil {
		ia.r.GetLog().Error(err, "Failed to get FCG Image Analyzer ConfigMap")
		return false, err
	}

	if !reflect.DeepEqual(cm.Data, existing.Data) {
		existing.Data = cm.Data
		return true, k8sutils.Update(ia.r, ctx, ia.cfg.Request, ia.r.GetLog(), ia.cfg.Owner, ia.cfg.Status, existing)
	}
	return false, nil
}

func (ia *ImageAnalyzer) buildConfigMap() (*corev1.ConfigMap, error) {
	spec := ia.cfg.ImageAnalyzerSpec
	data := map[string]string{}

	data["FALCON_MODE"] = "iar"
	// AGENT_HELM_VERSION must be >= 1.1.17 for latest IAR features
	data["AGENT_HELM_VERSION"] = "1.1.17"

	if ia.cfg.FalconAPI != nil {
		if ia.cfg.FalconAPI.ClientId != "" {
			data["AGENT_CLIENT_ID"] = ia.cfg.FalconAPI.ClientId
		}
		if ia.cfg.FalconAPI.ClientSecret != "" {
			data["AGENT_CLIENT_SECRET"] = ia.cfg.FalconAPI.ClientSecret
		}
		data["AGENT_REGION"] = falcon.Cloud(ia.cfg.FalconAPI.CloudRegion).String()
	}

	if spec.ClusterName != "" {
		data["AGENT_CLUSTER_NAME"] = spec.ClusterName
	}

	if len(spec.RegistryConfig.Credentials) > 0 {
		for _, v := range spec.RegistryConfig.Credentials {
			data["AGENT_REGISTRY_CREDENTIALS"] = fmt.Sprintf("%s:%s", v.Namespace, v.SecretName)
		}
	}

	if len(spec.Exclusions.Namespaces) > 0 {
		data["AGENT_NAMESPACE_EXCLUSIONS"] = strings.Join(spec.Exclusions.Namespaces, ",")
	}
	if len(spec.Exclusions.Registries) > 0 {
		data["AGENT_REGISTRY_EXCLUSIONS"] = strings.Join(spec.Exclusions.Registries, ",")
	}
	if len(spec.Exclusions.ImageNames) > 0 {
		data["AGENT_IMAGE_EXCLUSIONS"] = strings.Join(spec.Exclusions.ImageNames, ",")
	}

	data["AGENT_DEBUG"] = strconv.FormatBool(spec.EnableDebug)
	data["LOG_VERBOSITY"] = spec.LogVerbosity
	data["SECRETS_AUTODISCOVER_ENABLED"] = strconv.FormatBool(spec.RegistryConfig.AutoDiscoverCredentials)
	data["IAR_AGENT_SERVICE_PORT"] = strconv.Itoa(int(spec.IARAgentService.Port))

	// KAC is always co-located with IAR in FCG; use the shared install namespace
	data["__CS_KAC_NAMESPACE"] = ia.cfg.InstallNamespace

	data["IS_KUBERNETES"] = isKubernetes
	data["AGENT_CID"] = ia.cfg.Cid
	data["AGENT_RUNMODE"] = agentRunmode
	data["AGENT_MAX_CONSUMER_THREADS"] = agentMaxConsumerThreads

	data["AGENT_TEMP_MOUNT_SIZE"] = "20Gi"
	if spec.VolumeSizeLimit != "" {
		data["AGENT_TEMP_MOUNT_SIZE"] = spec.VolumeSizeLimit
	}

	return assets.SensorConfigMap(pkgcommon.FCGImageAnalyzerConfigMapName, ia.cfg.InstallNamespace, pkgcommon.FCGImageAnalyzerComponentName, data), nil
}
