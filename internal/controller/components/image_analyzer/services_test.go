package image_analyzer

import (
	"context"
	"testing"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	pkgcommon "github.com/crowdstrike/falcon-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// KAC discovers IAR by these labels, so they must match the standalone FalconImageAnalyzer values.
func assertKACDiscoveryLabels(t *testing.T, obj client.Object) {
	t.Helper()
	labels := obj.GetLabels()
	if got := labels[pkgcommon.KubernetesComponentKey]; got != pkgcommon.FalconImageAnalyzerComponentName {
		t.Errorf("expected %s=%q, got %q", pkgcommon.KubernetesComponentKey, pkgcommon.FalconImageAnalyzerComponentName, got)
	}
	if got := labels[pkgcommon.AppLabelKey]; got != pkgcommon.FalconImageAnalyzerAgentServiceApp {
		t.Errorf("expected %s=%q, got %q", pkgcommon.AppLabelKey, pkgcommon.FalconImageAnalyzerAgentServiceApp, got)
	}
}

func TestReconcileService_KACDiscoveryLabels(t *testing.T) {
	owner := ownerCR()
	r := newFakeReconciler(owner)
	ia := newTestIA(r, owner, falconv1alpha1.FalconClusterGuardImageAnalyzerSpec{})
	ctx := context.Background()

	if err := ia.reconcileService(ctx); err != nil {
		t.Fatalf("reconcileService() error: %v", err)
	}
	svc := &corev1.Service{}
	if err := r.Get(ctx, types.NamespacedName{Name: pkgcommon.FalconImageAnalyzerAgentService, Namespace: testNamespace}, svc); err != nil {
		t.Fatalf("Get Service: %v", err)
	}
	assertKACDiscoveryLabels(t, svc)
}

func TestReconcileTLSSecret_KACDiscoveryLabels(t *testing.T) {
	owner := ownerCR()
	r := newFakeReconciler(owner)
	ia := newTestIA(r, owner, falconv1alpha1.FalconClusterGuardImageAnalyzerSpec{})
	ctx := context.Background()

	if err := ia.reconcileTLSSecret(ctx); err != nil {
		t.Fatalf("reconcileTLSSecret() error: %v", err)
	}
	secret := &corev1.Secret{}
	if err := r.Get(ctx, types.NamespacedName{Name: pkgcommon.FCGImageAnalyzerTLSSecretName, Namespace: testNamespace}, secret); err != nil {
		t.Fatalf("Get Secret: %v", err)
	}
	assertKACDiscoveryLabels(t, secret)
}

func TestReconcileServiceAccount_OpenShiftPullSecretsPreserved(t *testing.T) {
	for _, openShift := range []bool{true, false} {
		owner := ownerCR()
		r := newFakeReconciler(owner)
		ia := newTestIA(r, owner, falconv1alpha1.FalconClusterGuardImageAnalyzerSpec{})
		ia.cfg.OpenShift = openShift
		ctx := context.Background()
		key := types.NamespacedName{Name: pkgcommon.FCGImageAnalyzerServiceAccountName, Namespace: testNamespace}

		if err := ia.reconcileServiceAccount(ctx); err != nil {
			t.Fatalf("reconcileServiceAccount() error: %v", err)
		}

		// Simulate OpenShift injecting its dockercfg pull secret.
		sa := &corev1.ServiceAccount{}
		if err := r.Get(ctx, key, sa); err != nil {
			t.Fatalf("Get ServiceAccount: %v", err)
		}
		injected := corev1.LocalObjectReference{Name: pkgcommon.FCGImageAnalyzerServiceAccountName + "-dockercfg-abcde"}
		sa.ImagePullSecrets = append(sa.ImagePullSecrets, injected)
		if err := r.Update(ctx, sa); err != nil {
			t.Fatalf("Update ServiceAccount: %v", err)
		}
		injectedRV := sa.ResourceVersion

		if err := ia.reconcileServiceAccount(ctx); err != nil {
			t.Fatalf("reconcileServiceAccount() error: %v", err)
		}
		got := &corev1.ServiceAccount{}
		if err := r.Get(ctx, key, got); err != nil {
			t.Fatalf("Get ServiceAccount: %v", err)
		}

		kept := len(got.ImagePullSecrets) == 1 && got.ImagePullSecrets[0] == injected
		if openShift && (!kept || got.ResourceVersion != injectedRV) {
			t.Errorf("OpenShift: expected injected pull secret kept without update, got %v (rv %q -> %q)",
				got.ImagePullSecrets, injectedRV, got.ResourceVersion)
		}
		if !openShift && kept {
			t.Errorf("non-OpenShift: expected unmanaged pull secret removed, got %v", got.ImagePullSecrets)
		}
	}
}
