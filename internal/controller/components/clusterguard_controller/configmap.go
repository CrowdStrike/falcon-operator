package clusterguard_controller

import (
	"context"
	"reflect"
	"strconv"

	"github.com/crowdstrike/falcon-operator/internal/controller/assets"
	k8sutils "github.com/crowdstrike/falcon-operator/internal/controller/common"
	pkgcommon "github.com/crowdstrike/falcon-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
)

// The admission controller process itself creates and manages the following ConfigMaps at runtime:
//   - falcon-kac-aid   – stores the sensor AID (agent ID) after registration
//   - falcon-kac-meta  – stores cluster metadata (e.g. ClusterName discovered by the sensor)
//   - falcon-kac-pol   – stores policy data pulled from the Falcon platform
//   - falcon-kac-state – stores internal sensor state
//
// These are not managed by the operator; do not attempt to reconcile or delete them.

// configMap builds the ConfigMap for the FalconClusterGuard admission controller.
func (a *ClusterGuardController) configMap() *corev1.ConfigMap {
	cfg := a.cfg.ClusterGuardControllerConfig

	// Start with sensor environment variables from the Falcon sensor config
	data := pkgcommon.MakeSensorEnvMap(a.cfg.Falcon)

	// Default FALCONCTL_OPT_TRACE to warn if not explicitly configured
	if _, set := data["FALCONCTL_OPT_TRACE"]; !set {
		data["FALCONCTL_OPT_TRACE"] = "warn"
	}

	// Add admission-specific configuration
	data["FALCON_MODE"] = "kac"
	data["WEBHOOK_PORT"] = pkgcommon.AdmissionWebhookPortStr
	data["GRPC_PORT"] = pkgcommon.AdmissionGRPCPortStr
	data["WATCHER_HTTP_PORT"] = pkgcommon.AdmissionWatcherHTTPPortStr
	data["__CS_ADMISSION_CONTROL_ENABLED"] = strconv.FormatBool(cfg.AdmissionControlEnabled == nil || *cfg.AdmissionControlEnabled)
	data["__CS_WATCH_EVENTS_ENABLED"] = strconv.FormatBool(cfg.GetWatcherEnabled())
	data["__CS_VISIBILITY_CONFIGMAPS_ENABLED"] = strconv.FormatBool(cfg.GetConfigMapWatcherEnabled())
	data["__CS_SNAPSHOTS_ENABLED"] = strconv.FormatBool(cfg.GetSnapshotsEnabled())
	data["__CS_SNAPSHOT_INTERVAL"] = cfg.GetSnapshotsInterval().String()
	data["__CS_GRPC_API_ENABLED"] = "true"
	data["FALCONCTL_OPT_CID"] = a.cfg.Cid

	return assets.SensorConfigMap(a.prefix()+"-config", a.cfg.InstallNamespace, pkgcommon.AdmissionComponentName, data)
}

// syncConfigMap creates or updates a ConfigMap when its Data has drifted.
// It returns true if the ConfigMap data was updated.
func (a *ClusterGuardController) syncConfigMap(ctx context.Context, cm *corev1.ConfigMap, logLabel string) (bool, error) {
	existing := &corev1.ConfigMap{}
	found, err := k8sutils.GetOrCreate(ctx, a.r, a.cfg.Request, a.cfg.Owner, a.cfg.Status, cm, existing,
		types.NamespacedName{Name: cm.Name, Namespace: cm.Namespace},
		"Failed to get "+logLabel+" ConfigMap")
	if !found || err != nil {
		return false, err
	}
	if !reflect.DeepEqual(cm.Data, existing.Data) {
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			if err := pkgcommon.GetNamespacedObject(ctx, a.r, a.r.GetK8sReader(),
				types.NamespacedName{Name: cm.Name, Namespace: cm.Namespace}, existing); err != nil {
				return err
			}
			for k, v := range cm.Data {
				if existing.Data[k] != v {
					a.r.GetLog().V(1).Info("Updating "+logLabel+" ConfigMap: value changed", "key", k, "old", existing.Data[k], "new", v)
				}
			}
			existing.Data = cm.Data
			existing.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ConfigMap"))
			return k8sutils.Update(a.r, ctx, a.cfg.Request, a.r.GetLog(), a.cfg.Owner, a.cfg.Status, existing)
		})
		return err == nil, err
	}
	return false, nil
}

// reconcileConfigMap reconciles the admission controller ConfigMap.
func (a *ClusterGuardController) reconcileConfigMap(ctx context.Context) (bool, error) {
	cm := a.configMap()
	return a.syncConfigMap(ctx, cm, "FalconClusterGuard")
}

// clusterNameConfigMap builds the cluster name ConfigMap.
func (a *ClusterGuardController) clusterNameConfigMap() *corev1.ConfigMap {
	return assets.SensorConfigMap(
		pkgcommon.FalconAdmissionClusterNameConfigMapName,
		a.cfg.InstallNamespace,
		pkgcommon.AdmissionComponentName,
		map[string]string{"ClusterName": *a.cfg.ClusterName},
	)
}

// reconcileClusterNameConfigMap creates or updates the cluster name ConfigMap when
// ClusterName is set, or removes the ClusterName key when it is unset.
func (a *ClusterGuardController) reconcileClusterNameConfigMap(ctx context.Context) (bool, error) {
	if a.cfg.ClusterName == nil {
		return a.removeClusterNameConfigMapKey(ctx)
	}
	return a.syncConfigMap(ctx, a.clusterNameConfigMap(), "FalconClusterGuard ClusterName")
}

// removeClusterNameConfigMapKey deletes the ClusterName key from the cluster name
// ConfigMap if it exists.
func (a *ClusterGuardController) removeClusterNameConfigMapKey(ctx context.Context) (bool, error) {
	existing := &corev1.ConfigMap{}
	err := pkgcommon.GetNamespacedObject(ctx, a.r, a.r.GetK8sReader(),
		types.NamespacedName{
			Name:      pkgcommon.FalconAdmissionClusterNameConfigMapName,
			Namespace: a.cfg.InstallNamespace,
		}, existing)
	if err != nil {
		return false, nil
	}
	if _, exists := existing.Data["ClusterName"]; !exists {
		return false, nil
	}
	delete(existing.Data, "ClusterName")
	existing.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ConfigMap"))
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		return k8sutils.Update(a.r, ctx, a.cfg.Request, a.r.GetLog(), a.cfg.Owner, a.cfg.Status, existing)
	})
	return err == nil, err
}
