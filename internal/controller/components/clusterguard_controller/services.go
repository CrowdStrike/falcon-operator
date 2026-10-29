package clusterguard_controller

import (
	"context"
	"maps"
	"reflect"

	"github.com/crowdstrike/falcon-operator/internal/controller/assets"
	k8sutils "github.com/crowdstrike/falcon-operator/internal/controller/common"
	pkgcommon "github.com/crowdstrike/falcon-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
)

// webhookService builds the Service that exposes the admission webhook.
func (a *ClusterGuardController) webhookService() *corev1.Service {
	selector := map[string]string{"app": pkgcommon.ClusterGuardServiceApp}
	// The image analyzer discovers the admission webhook Service by app=falcon-kac, so the Service's own
	// app label differs from the selector, which matches the clusterguard controller pods.
	labels := a.withAppLabels(map[string]string{"app": pkgcommon.FalconAdmissionServiceApp})

	port := int32(443)
	if a.cfg.ClusterGuardControllerConfig.Port != nil {
		port = *a.cfg.ClusterGuardControllerConfig.Port
	}

	return assets.ServiceWithCustomLabels(
		pkgcommon.ClusterGuardControllerWebhookServiceName,
		a.cfg.InstallNamespace,
		selector,
		labels,
		pkgcommon.FalconServiceHTTPSName,
		"webhook-port",
		port,
	)
}

// apiService builds the Service that exposes the gRPC API.
func (a *ClusterGuardController) apiService() *corev1.Service {
	selector := map[string]string{"app": pkgcommon.ClusterGuardServiceApp}
	labels := a.withAppLabels(map[string]string{"app": pkgcommon.ClusterGuardServiceApp})

	return assets.ServiceWithCustomLabels(
		pkgcommon.ClusterGuardControllerAPIServiceName,
		a.cfg.InstallNamespace,
		selector,
		labels,
		"grpc",
		"grpc-port",
		pkgcommon.FalconServiceHTTPSPort,
	)
}

// syncService creates the Service or updates it when its ports, selector or labels have drifted.
// It returns true only when the ports or selector changed, which requires a pod restart.
func (a *ClusterGuardController) syncService(ctx context.Context, svc *corev1.Service) (bool, error) {
	key := types.NamespacedName{Name: svc.Name, Namespace: svc.Namespace}
	existing := &corev1.Service{}
	found, err := k8sutils.GetOrCreate(ctx, a.r, a.cfg.Request, a.cfg.Owner, a.cfg.Status, svc, existing, key,
		"Failed to get FalconClusterGuard "+svc.Name+" Service")
	if !found || err != nil {
		return false, err
	}
	specChanged := !reflect.DeepEqual(svc.Spec.Ports, existing.Spec.Ports) || !reflect.DeepEqual(svc.Spec.Selector, existing.Spec.Selector)
	if !specChanged && !labelsDrifted(svc.Labels, existing.Labels) {
		return false, nil
	}
	err = retry.RetryOnConflict(retry.DefaultRetry, func() error {
		if err := pkgcommon.GetNamespacedObject(ctx, a.r, a.r.GetK8sReader(), key, existing); err != nil {
			return err
		}
		a.r.GetLog().V(1).Info("Updating FalconClusterGuard Service: ports, selector or labels changed", "service", svc.Name)
		existing.Spec.Ports = svc.Spec.Ports
		existing.Spec.Selector = svc.Spec.Selector
		if existing.Labels == nil {
			existing.Labels = map[string]string{}
		}
		maps.Copy(existing.Labels, svc.Labels)
		existing.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Service"))
		return k8sutils.Update(a.r, ctx, a.cfg.Request, a.r.GetLog(), a.cfg.Owner, a.cfg.Status, existing)
	})
	return err == nil && specChanged, err
}

// labelsDrifted reports whether any desired label is missing from existing or has a different value.
func labelsDrifted(desired, existing map[string]string) bool {
	for k, v := range desired {
		if existing[k] != v {
			return true
		}
	}
	return false
}

// reconcileWebhookService returns true if the service ports or selector changed, which requires a pod restart.
func (a *ClusterGuardController) reconcileWebhookService(ctx context.Context) (bool, error) {
	return a.syncService(ctx, a.webhookService())
}

// reconcileAPIService returns true if the service ports or selector changed, which requires a pod restart.
func (a *ClusterGuardController) reconcileAPIService(ctx context.Context) (bool, error) {
	return a.syncService(ctx, a.apiService())
}
