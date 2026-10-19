package node_sensor

import (
	"context"
	"reflect"

	k8sutils "github.com/crowdstrike/falcon-operator/internal/controller/common"
	pkgcommon "github.com/crowdstrike/falcon-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/util/retry"
)

// proxyService builds the node-local Service that exposes the guardian proxy port on each node.
// internalTrafficPolicy: Local ensures traffic is only routed to the pod on the same node.
func (n *NodeSensor) proxyService() *corev1.Service {
	internalTrafficPolicy := corev1.ServiceInternalTrafficPolicyLocal
	selector := map[string]string{
		pkgcommon.KubernetesComponentKey: pkgcommon.ClusterGuardNodeSensorComponentName,
	}
	labels := pkgcommon.CRLabels("service", pkgcommon.ClusterGuardProxyServiceName, pkgcommon.ClusterGuardNodeSensorComponentName)

	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      pkgcommon.ClusterGuardProxyServiceName,
			Namespace: n.cfg.InstallNamespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Selector:              selector,
			InternalTrafficPolicy: &internalTrafficPolicy,
			Ports: []corev1.ServicePort{
				{
					Name:       pkgcommon.ClusterGuardProxyPortName,
					Port:       pkgcommon.ClusterGuardProxyServicePort,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromString(pkgcommon.ClusterGuardProxyPortName),
				},
			},
		},
	}
}

// reconcileProxyService creates or updates the shared falcon-proxy Service when the guardian
// proxy is enabled. The Service is static — shared across all node sensor DaemonSets in the
// namespace — so deletion is handled at the FalconClusterGuard level, not here.
func (n *NodeSensor) reconcileProxyService(ctx context.Context) error {
	if !n.cfg.NodeSensor.Guardian.Proxy.IsEnabled() {
		return nil
	}

	svc := n.proxyService()

	existing := &corev1.Service{}
	found, err := k8sutils.GetOrCreate(ctx, n.r, n.cfg.Request, n.cfg.Owner, n.cfg.Status, svc, existing,
		types.NamespacedName{Name: pkgcommon.ClusterGuardProxyServiceName, Namespace: n.cfg.InstallNamespace},
		"Failed to create FalconClusterGuard proxy Service")
	if !found || err != nil {
		return err
	}

	if !reflect.DeepEqual(svc.Spec.Ports, existing.Spec.Ports) ||
		!reflect.DeepEqual(svc.Spec.Selector, existing.Spec.Selector) ||
		!reflect.DeepEqual(svc.Spec.InternalTrafficPolicy, existing.Spec.InternalTrafficPolicy) {
		return retry.RetryOnConflict(retry.DefaultRetry, func() error {
			if err := pkgcommon.GetNamespacedObject(ctx, n.r, n.r.GetK8sReader(),
				types.NamespacedName{Name: pkgcommon.ClusterGuardProxyServiceName, Namespace: n.cfg.InstallNamespace},
				existing); err != nil {
				return err
			}
			log := n.r.GetLog()
			if !reflect.DeepEqual(svc.Spec.Ports, existing.Spec.Ports) {
				log.V(1).Info("Updating FalconClusterGuard proxy Service: ports changed",
					"old", existing.Spec.Ports, "new", svc.Spec.Ports)
			}
			if !reflect.DeepEqual(svc.Spec.Selector, existing.Spec.Selector) {
				log.V(1).Info("Updating FalconClusterGuard proxy Service: selector changed",
					"old", existing.Spec.Selector, "new", svc.Spec.Selector)
			}
			if !reflect.DeepEqual(svc.Spec.InternalTrafficPolicy, existing.Spec.InternalTrafficPolicy) {
				log.V(1).Info("Updating FalconClusterGuard proxy Service: internalTrafficPolicy changed",
					"old", existing.Spec.InternalTrafficPolicy, "new", svc.Spec.InternalTrafficPolicy)
			}
			existing.Spec.Ports = svc.Spec.Ports
			existing.Spec.Selector = svc.Spec.Selector
			existing.Spec.InternalTrafficPolicy = svc.Spec.InternalTrafficPolicy
			existing.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Service"))
			return k8sutils.Update(n.r, ctx, n.cfg.Request, log, n.cfg.Owner, n.cfg.Status, existing)
		})
	}
	return nil
}
