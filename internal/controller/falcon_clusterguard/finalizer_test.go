package controllers

import (
	"context"
	"slices"
	"testing"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
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

// TestReconcile_DeletionRunsNodeCleanupWithoutSetup checks that deletion does not depend on the setup in Reconcile:
// the CR has no CID or Falcon API credentials, so resolving the CID would fail. Where the cleanup image comes from is
// covered by the node_sensor tests.
func TestReconcile_DeletionRunsNodeCleanupWithoutSetup(t *testing.T) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = falconv1alpha1.AddToScheme(scheme)
	const namespace = "falcon-clusterguard"

	fcg := &falconv1alpha1.FalconClusterGuard{
		ObjectMeta: metav1.ObjectMeta{Name: "test-fcg", UID: "uid-test-fcg", Finalizers: []string{common.FalconFinalizer},
			DeletionTimestamp: ptr.To(metav1.Now())},
		Spec: falconv1alpha1.FalconClusterGuardSpec{InstallNamespace: namespace},
	}
	sensor := &appsv1.DaemonSet{
		ObjectMeta: metav1.ObjectMeta{Name: "falcon-sensor", Namespace: namespace, OwnerReferences: []metav1.OwnerReference{{
			APIVersion: falconv1alpha1.GroupVersion.String(), Kind: "FalconClusterGuard", Name: fcg.Name, UID: fcg.UID, Controller: ptr.To(true),
		}}},
		Spec: appsv1.DaemonSetSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{
			Name: "falcon-node-sensor", Image: "registry.example.com/falcon-sensor:1.0", ImagePullPolicy: corev1.PullAlways,
		}}}}},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(fcg, sensor).WithStatusSubresource(fcg).Build()
	r := &FalconClusterGuardReconciler{Client: c, Reader: c, RuntimeScheme: scheme}

	result, err := r.Reconcile(ctx, ctrl.Request{NamespacedName: types.NamespacedName{Name: fcg.Name}})
	if err != nil {
		t.Fatalf("Reconcile() error: %v", err)
	}
	if result.RequeueAfter == 0 {
		t.Error("expected Reconcile to requeue while cleanup pods are pending")
	}
	if err := c.Get(ctx, types.NamespacedName{Name: "falcon-sensor-cleanup", Namespace: namespace}, &appsv1.DaemonSet{}); err != nil {
		t.Errorf("expected the cleanup DaemonSet to be created: %v", err)
	}
}
