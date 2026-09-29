package node_sensor

import (
	"context"
	"fmt"
	"reflect"

	"github.com/crowdstrike/falcon-operator/internal/controller/assets"
	k8sutils "github.com/crowdstrike/falcon-operator/internal/controller/common"
	pkgcommon "github.com/crowdstrike/falcon-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
)

// configMap builds the ConfigMap for the node sensor.
// The name is a fixed cluster-wide constant — see rbac.go serviceAccount() for rationale.
func (n *NodeSensor) configMap() *corev1.ConfigMap {
	apiServiceName := fmt.Sprintf("%s.%s.svc", pkgcommon.ClusterGuardAPIServiceName, n.cfg.InstallNamespace)

	// Start with sensor environment variables from the Falcon sensor config
	data := pkgcommon.MakeSensorEnvMap(n.cfg.Falcon)

	// Default FALCONCTL_OPT_TRACE to info if not explicitly configured
	if _, set := data["FALCONCTL_OPT_TRACE"]; !set {
		data["FALCONCTL_OPT_TRACE"] = "info"
	}

	// Add node sensor-specific configuration
	data["FALCONCTL_OPT_BACKEND"] = "bpf"
	data["FALCON_MODE"] = "daemonset"
	data["__CS_ENABLE_K8S_METADATA_SERVICE"] = "true"
	data["API_SERVICE_NAME"] = apiServiceName

	if n.cfg.NodeSensor.ClusterName != nil {
		data["FALCON_CLUSTER_NAME"] = *n.cfg.NodeSensor.ClusterName
	}

	// CID can be overridden by the controller
	if n.cfg.Cid != "" {
		data["FALCONCTL_OPT_CID"] = n.cfg.Cid
	}

	return assets.SensorConfigMap(pkgcommon.ClusterGuardSensorConfigMapName, n.cfg.InstallNamespace, pkgcommon.ClusterGuardComponentName, data)
}

// syncConfigMap creates or updates a ConfigMap when its Data has drifted.
func (n *NodeSensor) syncConfigMap(ctx context.Context, cm *corev1.ConfigMap, logLabel string) error {
	existing := &corev1.ConfigMap{}
	found, err := k8sutils.GetOrCreate(ctx, n.r, n.cfg.Request, n.cfg.Owner, n.cfg.Status, cm, existing,
		types.NamespacedName{Name: cm.Name, Namespace: cm.Namespace},
		"Failed to get "+logLabel+" ConfigMap")
	if !found || err != nil {
		return err
	}
	if !reflect.DeepEqual(cm.Data, existing.Data) {
		err := retry.RetryOnConflict(retry.DefaultRetry, func() error {
			if err := pkgcommon.GetNamespacedObject(ctx, n.r, n.r.GetK8sReader(),
				types.NamespacedName{Name: cm.Name, Namespace: cm.Namespace}, existing); err != nil {
				return err
			}
			for k, v := range cm.Data {
				if existing.Data[k] != v {
					n.r.GetLog().V(1).Info("Updating "+logLabel+" ConfigMap: value changed", "key", k, "old", existing.Data[k], "new", v)
				}
			}
			existing.Data = cm.Data
			existing.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ConfigMap"))
			return k8sutils.Update(n.r, ctx, n.cfg.Request, n.r.GetLog(), n.cfg.Owner, n.cfg.Status, existing)
		})
		return err
	}
	return nil
}

func (n *NodeSensor) reconcileConfigMap(ctx context.Context) error {
	cm := n.configMap()
	return n.syncConfigMap(ctx, cm, "FalconClusterGuard sensor")
}
