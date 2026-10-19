package image_analyzer

import (
	"context"
	"reflect"

	"github.com/crowdstrike/falcon-operator/internal/controller/assets"
	k8sutils "github.com/crowdstrike/falcon-operator/internal/controller/common"
	pkgcommon "github.com/crowdstrike/falcon-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (ia *ImageAnalyzer) reconcileService(ctx context.Context) error {
	selector := map[string]string{pkgcommon.FalconComponentKey: pkgcommon.FCGImageAnalyzerComponentName}

	// Labels required for KAC -> IAR communication; values must match the standalone FalconImageAnalyzer.
	labels := pkgcommon.CRLabels("service", pkgcommon.FCGImageAnalyzerDeploymentName, pkgcommon.FCGImageAnalyzerComponentName)
	labels[pkgcommon.AppLabelKey] = pkgcommon.FalconImageAnalyzerAgentServiceApp
	labels[pkgcommon.KubernetesComponentKey] = pkgcommon.FalconImageAnalyzerComponentName
	labels[pkgcommon.KubernetesNameKey] = pkgcommon.FCGImageAnalyzerDeploymentName

	service := assets.ServiceWithCustomLabels(
		pkgcommon.FalconImageAnalyzerAgentService,
		ia.cfg.InstallNamespace,
		selector,
		labels,
		"",
		pkgcommon.FalconImageAnalyzerAgentServicePortName,
		pkgcommon.FalconImageAnalyzerAgentServicePort,
	)

	existing := &corev1.Service{}
	found, err := k8sutils.GetOrCreate(ctx, ia.r, ia.cfg.Request, ia.cfg.Owner, ia.cfg.Status, service, existing,
		types.NamespacedName{Name: pkgcommon.FalconImageAnalyzerAgentService, Namespace: ia.cfg.InstallNamespace},
		"Failed to get FCG Image Analyzer Agent Service")
	if !found || err != nil {
		return err
	}

	updated := false
	if !reflect.DeepEqual(service.Spec.Ports, existing.Spec.Ports) {
		existing.Spec.Ports = service.Spec.Ports
		updated = true
	}
	if !reflect.DeepEqual(service.Spec.Selector, existing.Spec.Selector) {
		existing.Spec.Selector = service.Spec.Selector
		updated = true
	}
	if existing.ObjectMeta.Labels == nil {
		existing.ObjectMeta.Labels = make(map[string]string)
	}
	for key, value := range service.ObjectMeta.Labels {
		if existing.ObjectMeta.Labels[key] != value {
			existing.ObjectMeta.Labels[key] = value
			updated = true
		}
	}
	if updated {
		existing.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Service"))
		return k8sutils.Update(ia.r, ctx, ia.cfg.Request, ia.r.GetLog(), ia.cfg.Owner, ia.cfg.Status, existing)
	}
	return nil
}
