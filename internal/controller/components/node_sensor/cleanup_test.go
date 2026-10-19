package node_sensor

import (
	"context"
	"fmt"
	"testing"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	pkgcommon "github.com/crowdstrike/falcon-operator/pkg/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	schedulingv1 "k8s.io/api/scheduling/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const cleanupTestNamespace = "falcon-clusterguard"

func boolPtr(b bool) *bool { return &b }

func newCleanupTestSensor(t *testing.T, spec falconv1alpha1.FalconClusterGuardNodeSpec) (*fakeReconciler, *falconv1alpha1.FalconClusterGuard, *NodeSensor) {
	t.Helper()
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = falconv1alpha1.AddToScheme(scheme)

	owner := ownerCR()
	owner.UID = "uid-test-fcg"
	r := &fakeReconciler{Client: fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(owner).WithStatusSubresource(owner).Build()}
	cfg := baseConfig(cleanupTestNamespace, owner)
	cfg.Status = &owner.Status
	cfg.Cid = "abc123-xx"
	cfg.NodeSensor = spec
	return r, owner, New(r, cfg)
}

// controlledBy lists every object of the kinds the node sensor creates that is controlled by owner.
func controlledBy(t *testing.T, c client.Client, owner client.Object) []string {
	t.Helper()
	lists := []client.ObjectList{
		&appsv1.DaemonSetList{}, &corev1.ConfigMapList{}, &corev1.ServiceAccountList{},
		&rbacv1.ClusterRoleBindingList{}, &schedulingv1.PriorityClassList{},
	}
	var names []string
	for _, l := range lists {
		if err := c.List(context.Background(), l); err != nil {
			t.Fatal(err)
		}
		items, err := meta.ExtractList(l)
		if err != nil {
			t.Fatal(err)
		}
		for _, item := range items {
			obj := item.(client.Object)
			if metav1.IsControlledBy(obj, owner) {
				names = append(names, fmt.Sprintf("%T/%s", obj, obj.GetName()))
			}
		}
	}
	return names
}

func getPersistedOwner(t *testing.T, r *fakeReconciler, owner *falconv1alpha1.FalconClusterGuard) *falconv1alpha1.FalconClusterGuard {
	t.Helper()
	persisted := &falconv1alpha1.FalconClusterGuard{}
	if err := r.Get(context.Background(), types.NamespacedName{Name: owner.Name}, persisted); err != nil {
		t.Fatal(err)
	}
	return persisted
}

func dsExists(t *testing.T, r *fakeReconciler, name string) bool {
	t.Helper()
	err := r.Get(context.Background(), types.NamespacedName{Name: name, Namespace: cleanupTestNamespace}, &appsv1.DaemonSet{})
	return err == nil
}

func TestReconcile_PersistsReadyConditionWithoutFinalizer(t *testing.T) {
	r, owner, n := newCleanupTestSensor(t, falconv1alpha1.FalconClusterGuardNodeSpec{})

	if _, err := n.Reconcile(context.Background()); err != nil {
		t.Fatalf("Reconcile() error: %v", err)
	}
	persisted := getPersistedOwner(t, r, owner)
	if !meta.IsStatusConditionTrue(persisted.Status.Conditions, falconv1alpha1.ConditionNodeSensorReady) {
		t.Errorf("expected %s condition to be persisted, got %v", falconv1alpha1.ConditionNodeSensorReady, persisted.Status.Conditions)
	}
	if len(persisted.Finalizers) != 0 {
		t.Errorf("finalizers are managed by the FalconClusterGuard controller, got %v", persisted.Finalizers)
	}
}

func TestCleanup_DisableCleanupDeletesWithoutNodeCleanup(t *testing.T) {
	ctx := context.Background()
	r, owner, n := newCleanupTestSensor(t, falconv1alpha1.FalconClusterGuardNodeSpec{
		NodeCleanup:   boolPtr(true),
		PriorityClass: falconv1alpha1.FalconClusterGuardPriorityClassConfig{Deploy: boolPtr(true)},
	})
	if _, err := n.Reconcile(ctx); err != nil {
		t.Fatalf("Reconcile() error: %v", err)
	}

	for i := range 2 {
		result, err := n.Cleanup(ctx)
		if err != nil {
			t.Fatalf("call %d: Cleanup() error: %v", i+1, err)
		}
		if result.RequeueAfter > 0 {
			t.Fatalf("call %d: Cleanup() should not requeue when node cleanup is disabled", i+1)
		}
	}

	if dsExists(t, r, "falcon-sensor-cleanup") {
		t.Error("cleanup DaemonSet must not be created when disableCleanup is true")
	}
	if left := controlledBy(t, r, owner); len(left) != 0 {
		t.Errorf("Cleanup left resources behind: %v", left)
	}
	if c := meta.FindStatusCondition(getPersistedOwner(t, r, owner).Status.Conditions, falconv1alpha1.ConditionNodeSensorReady); c != nil {
		t.Errorf("expected %s condition to be removed, got %v", falconv1alpha1.ConditionNodeSensorReady, c)
	}
}

func TestCleanup_RunsNodeCleanupBeforeDeleting(t *testing.T) {
	ctx := context.Background()
	r, owner, n := newCleanupTestSensor(t, falconv1alpha1.FalconClusterGuardNodeSpec{
		PriorityClass: falconv1alpha1.FalconClusterGuardPriorityClassConfig{Deploy: boolPtr(true)},
	})
	if _, err := n.Reconcile(ctx); err != nil {
		t.Fatalf("Reconcile() error: %v", err)
	}

	result, err := n.Cleanup(ctx)
	if err != nil {
		t.Fatalf("Cleanup() error: %v", err)
	}
	if result.RequeueAfter == 0 {
		t.Fatal("expected Cleanup to requeue while cleanup pods are pending")
	}
	if dsExists(t, r, "falcon-sensor") {
		t.Error("sensor DaemonSet should be deleted before node cleanup runs")
	}
	if !dsExists(t, r, "falcon-sensor-cleanup") {
		t.Fatal("expected cleanup DaemonSet to be created")
	}
	if c := meta.FindStatusCondition(getPersistedOwner(t, r, owner).Status.Conditions, falconv1alpha1.ConditionNodeSensorReady); c == nil {
		t.Error("condition should only be removed once cleanup completes")
	}

	// Simulate the cleanup pod running on the only node.
	cleanupDS := &appsv1.DaemonSet{}
	if err := r.Get(ctx, types.NamespacedName{Name: "falcon-sensor-cleanup", Namespace: cleanupTestNamespace}, cleanupDS); err != nil {
		t.Fatal(err)
	}
	cleanupDS.Status.DesiredNumberScheduled = 1
	if err := r.Status().Update(ctx, cleanupDS); err != nil {
		t.Fatal(err)
	}
	if err := r.Create(ctx, &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "cleanup-pod", Namespace: cleanupTestNamespace,
			Labels: map[string]string{"app": "falcon-sensor-cleanup"}},
		Status: corev1.PodStatus{Phase: corev1.PodRunning},
	}); err != nil {
		t.Fatal(err)
	}

	for i := range 2 {
		result, err = n.Cleanup(ctx)
		if err != nil {
			t.Fatalf("call %d: Cleanup() error: %v", i+1, err)
		}
		if result.RequeueAfter > 0 {
			t.Fatalf("call %d: expected Cleanup to finish once cleanup pods are running", i+1)
		}
	}

	if left := controlledBy(t, r, owner); len(left) != 0 {
		t.Errorf("Cleanup left resources behind: %v", left)
	}
	if c := meta.FindStatusCondition(getPersistedOwner(t, r, owner).Status.Conditions, falconv1alpha1.ConditionNodeSensorReady); c != nil {
		t.Errorf("expected %s condition to be removed, got %v", falconv1alpha1.ConditionNodeSensorReady, c)
	}
}

func TestCleanup_NoSensorDaemonSetSkipsNodeCleanup(t *testing.T) {
	r, _, n := newCleanupTestSensor(t, falconv1alpha1.FalconClusterGuardNodeSpec{})

	result, err := n.Cleanup(context.Background())
	if err != nil {
		t.Fatalf("Cleanup() error: %v", err)
	}
	if result.RequeueAfter > 0 {
		t.Error("Cleanup should not requeue when the node sensor was never deployed")
	}
	if dsExists(t, r, "falcon-sensor-cleanup") {
		t.Error("cleanup DaemonSet must not be created when the node sensor was never deployed")
	}
}

func TestCleanup_KeepsDaemonSetNotControlledByOwner(t *testing.T) {
	ctx := context.Background()
	r, _, n := newCleanupTestSensor(t, falconv1alpha1.FalconClusterGuardNodeSpec{})
	if err := r.Create(ctx, &appsv1.DaemonSet{ObjectMeta: metav1.ObjectMeta{Name: "falcon-sensor", Namespace: cleanupTestNamespace}}); err != nil {
		t.Fatal(err)
	}

	if _, err := n.Cleanup(ctx); err != nil {
		t.Fatalf("Cleanup() error: %v", err)
	}
	if !dsExists(t, r, "falcon-sensor") {
		t.Error("DaemonSet not controlled by the FalconClusterGuard must be kept")
	}
	if dsExists(t, r, "falcon-sensor-cleanup") {
		t.Error("node cleanup must not run for a DaemonSet the FalconClusterGuard does not control")
	}
}

func TestCleanupDaemonSet_MatchesSensorSchedulingAndPullSecrets(t *testing.T) {
	tolerations := []corev1.Toleration{{Key: "node-role.kubernetes.io/control-plane", Effect: corev1.TaintEffectNoSchedule}}
	_, _, n := newCleanupTestSensor(t, falconv1alpha1.FalconClusterGuardNodeSpec{Tolerations: &tolerations})
	n.cfg.ImagePullSecrets = []corev1.LocalObjectReference{{Name: pkgcommon.FalconPullSecretName}}

	spec := n.cleanupDaemonSet().Spec.Template.Spec
	if len(spec.Tolerations) != 1 || spec.Tolerations[0].Key != tolerations[0].Key {
		t.Errorf("expected cleanup DaemonSet tolerations %v, got %v", tolerations, spec.Tolerations)
	}
	if len(spec.ImagePullSecrets) != 1 || spec.ImagePullSecrets[0].Name != pkgcommon.FalconPullSecretName {
		t.Errorf("expected cleanup DaemonSet pull secret %q, got %v", pkgcommon.FalconPullSecretName, spec.ImagePullSecrets)
	}
}
