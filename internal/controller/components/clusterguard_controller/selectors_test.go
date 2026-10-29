package clusterguard_controller

import (
	"context"
	"testing"

	"github.com/crowdstrike/falcon-operator/internal/controller/components"
	pkgcommon "github.com/crowdstrike/falcon-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
)

func TestSelectorsMatchControllerPods(t *testing.T) {
	a := New(nil, Config{BaseConfig: components.BaseConfig{InstallNamespace: cleanupTestNamespace}})
	dep := a.Deployment()
	podLabels := labels.Set(dep.Spec.Template.Labels)

	selectors := map[string]*metav1.LabelSelector{
		"Deployment":                   {MatchLabels: dep.Spec.Selector.MatchLabels},
		"topology spread":              dep.Spec.Template.Spec.TopologySpreadConstraints[0].LabelSelector,
		"webhook Service":              {MatchLabels: a.webhookService().Spec.Selector},
		"API Service":                  {MatchLabels: a.apiService().Spec.Selector},
		"ready pod lookup (Reconcile)": {MatchLabels: map[string]string{"app": pkgcommon.ClusterGuardServiceApp}},
	}
	for name, ls := range selectors {
		sel, err := metav1.LabelSelectorAsSelector(ls)
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		if !sel.Matches(podLabels) {
			t.Errorf("%s selector %v does not match controller pod labels %v", name, sel, podLabels)
		}
	}
}

// The image analyzer finds the admission webhook Service by app=falcon-kac, matching the Helm chart and the
// standalone FalconAdmission.
func TestWebhookServiceHasImageAnalyzerDiscoveryLabel(t *testing.T) {
	a := New(nil, Config{BaseConfig: components.BaseConfig{InstallNamespace: cleanupTestNamespace}})
	svc := a.webhookService()

	if got := svc.Labels["app"]; got != pkgcommon.FalconAdmissionServiceApp {
		t.Errorf("expected webhook Service label app=%q, got %q", pkgcommon.FalconAdmissionServiceApp, got)
	}
	if got := svc.Spec.Selector["app"]; got != pkgcommon.ClusterGuardServiceApp {
		t.Errorf("expected webhook Service selector app=%q, got %q", pkgcommon.ClusterGuardServiceApp, got)
	}
}

func TestReconcileWebhookServiceSyncsLabels(t *testing.T) {
	ctx := context.Background()
	desired := New(nil, Config{BaseConfig: components.BaseConfig{InstallNamespace: cleanupTestNamespace}}).webhookService()

	tests := []struct {
		name        string
		selector    map[string]string
		wantRestart bool
	}{
		{"labels only", desired.Spec.Selector, false},
		{"selector and labels", map[string]string{"app": "stale"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			existing := desired.DeepCopy()
			existing.Labels = map[string]string{"app": pkgcommon.ClusterGuardServiceApp, "user-label": "kept"}
			existing.Spec.Selector = tt.selector
			r, owner, a := newCleanupTestController(t, existing)

			restart, err := a.reconcileWebhookService(ctx)
			if err != nil {
				t.Fatal(err)
			}
			if restart != tt.wantRestart {
				t.Errorf("expected restart=%v, got %v", tt.wantRestart, restart)
			}

			got := &corev1.Service{}
			if err := r.Get(ctx, types.NamespacedName{Name: desired.Name, Namespace: desired.Namespace}, got); err != nil {
				t.Fatal(err)
			}
			want := map[string]string{
				"app":                           pkgcommon.FalconAdmissionServiceApp,
				pkgcommon.KubernetesInstanceKey: owner.Name,
				"user-label":                    "kept",
			}
			for k, v := range want {
				if got.Labels[k] != v {
					t.Errorf("expected label %s=%q, got %q", k, v, got.Labels[k])
				}
			}
			if got.Spec.Selector["app"] != pkgcommon.ClusterGuardServiceApp {
				t.Errorf("expected selector app=%q, got %v", pkgcommon.ClusterGuardServiceApp, got.Spec.Selector)
			}
		})
	}
}
