package controllers

import (
	"context"
	"slices"
	"testing"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/go-logr/logr"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestSetFinalizerDoesNotPersistInjectedSpec(t *testing.T) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	_ = falconv1alpha1.AddToScheme(scheme)

	stored := &falconv1alpha1.FalconClusterGuard{
		ObjectMeta: metav1.ObjectMeta{Name: "test-fcg"},
		Spec: falconv1alpha1.FalconClusterGuardSpec{
			FalconAPI: &falconv1alpha1.FalconAPI{CloudRegion: "autodiscover"},
		},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(stored).Build()
	r := &FalconClusterGuardReconciler{Client: c, Reader: c, RuntimeScheme: scheme, log: logr.Discard()}

	fcg := &falconv1alpha1.FalconClusterGuard{}
	if err := c.Get(ctx, types.NamespacedName{Name: "test-fcg"}, fcg); err != nil {
		t.Fatal(err)
	}
	// Simulate injectFalconSecretData populating credentials in memory only.
	fcg.Spec.FalconAPI.ClientId = "injected-id"
	fcg.Spec.FalconAPI.ClientSecret = "injected-secret"

	get := func() *falconv1alpha1.FalconClusterGuard {
		t.Helper()
		got := &falconv1alpha1.FalconClusterGuard{}
		if err := c.Get(ctx, types.NamespacedName{Name: "test-fcg"}, got); err != nil {
			t.Fatal(err)
		}
		return got
	}

	if err := r.setFinalizer(ctx, fcg, true); err != nil {
		t.Fatalf("setFinalizer(true) error: %v", err)
	}
	got := get()
	if !slices.Contains(got.Finalizers, common.FalconFinalizer) {
		t.Errorf("expected finalizer %q, got %v", common.FalconFinalizer, got.Finalizers)
	}
	if !slices.Contains(fcg.Finalizers, common.FalconFinalizer) {
		t.Error("expected in-memory finalizers to be updated")
	}
	if got.Spec.FalconAPI.ClientId != "" || got.Spec.FalconAPI.ClientSecret != "" {
		t.Errorf("injected credentials were persisted to the CR spec: %+v", got.Spec.FalconAPI)
	}
	if fcg.Spec.FalconAPI.ClientSecret != "injected-secret" {
		t.Error("setFinalizer must not reset the in-memory spec")
	}

	if err := r.setFinalizer(ctx, fcg, false); err != nil {
		t.Fatalf("setFinalizer(false) error: %v", err)
	}
	got = get()
	if slices.Contains(got.Finalizers, common.FalconFinalizer) {
		t.Errorf("expected finalizer %q to be removed, got %v", common.FalconFinalizer, got.Finalizers)
	}
	if got.Spec.FalconAPI.ClientId != "" || got.Spec.FalconAPI.ClientSecret != "" {
		t.Errorf("injected credentials were persisted to the CR spec: %+v", got.Spec.FalconAPI)
	}
}
