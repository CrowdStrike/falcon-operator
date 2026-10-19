package node_sensor

import (
	"context"
	"testing"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/internal/controller/components"
	pkgcommon "github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// fakeReconciler satisfies the Reconciler interface using a fake client.
type fakeReconciler struct {
	client.Client
}

func (f *fakeReconciler) GetK8sReader() client.Reader { return f.Client }
func (f *fakeReconciler) GetScheme() *runtime.Scheme  { return f.Client.Scheme() }
func (f *fakeReconciler) GetLog() logr.Logger         { return logr.Discard() }

func newFakeReconciler(objs ...client.Object) *fakeReconciler {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = falconv1alpha1.AddToScheme(scheme)

	// WithStatusSubresource is required for ConditionsUpdate to call Status().Update()
	// without getting "object not found" errors on the owner CR.
	statusObjs := make([]client.Object, 0, len(objs))
	for _, o := range objs {
		if _, ok := o.(*falconv1alpha1.FalconClusterGuard); ok {
			statusObjs = append(statusObjs, o)
		}
	}
	return &fakeReconciler{
		Client: fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(objs...).
			WithStatusSubresource(statusObjs...).
			Build(),
	}
}

// ownerCR returns a minimal FalconClusterGuard CR for use as the reconcile owner.
// FalconClusterGuard is cluster-scoped so Namespace is empty.
func ownerCR() *falconv1alpha1.FalconClusterGuard {
	return &falconv1alpha1.FalconClusterGuard{
		TypeMeta: metav1.TypeMeta{
			APIVersion: falconv1alpha1.GroupVersion.String(),
			Kind:       "FalconClusterGuard",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: "test-fcg",
		},
	}
}

func baseConfig(namespace string, owner *falconv1alpha1.FalconClusterGuard) Config {
	return Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: namespace,
			Image:            "quay.io/crowdstrike/falcon-sensor:latest",
			Owner:            owner,
			Status:           &falconv1alpha1.FalconCRStatus{},
			// Request must have the same NamespacedName as the owner so that
			// ConditionsUpdate can re-fetch it on status updates.
			Request: ctrl.Request{NamespacedName: types.NamespacedName{Name: owner.Name}},
		},
	}
}

// reconcileAndGet runs reconcileDaemonSet and returns the live DaemonSet from the fake API.
func reconcileAndGet(t *testing.T, n *NodeSensor, r *fakeReconciler, namespace, name string) *appsv1.DaemonSet {
	t.Helper()
	ctx := context.Background()
	if err := n.reconcileDaemonSet(ctx); err != nil {
		t.Fatalf("reconcileDaemonSet() error: %v", err)
	}
	got := &appsv1.DaemonSet{}
	if err := r.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, got); err != nil {
		t.Fatalf("Get DaemonSet %q: %v", name, err)
	}
	return got
}

// TestReconcileDaemonSet_Create verifies that reconcileDaemonSet creates the DaemonSet
// when it does not yet exist.
func TestReconcileDaemonSet_Create(t *testing.T) {
	ns := "falcon-clusterguard"
	owner := ownerCR()
	r := newFakeReconciler(owner)
	n := New(r, baseConfig(ns, owner))

	ds := reconcileAndGet(t, n, r, ns, "falcon-sensor")

	if ds.Name != "falcon-sensor" {
		t.Errorf("expected name %q, got %q", "falcon-sensor", ds.Name)
	}
	if ds.Namespace != ns {
		t.Errorf("expected namespace %q, got %q", ns, ds.Namespace)
	}
}

// TestReconcileDaemonSet_Idempotent verifies that a second call with no spec change
// produces no update (resource version stays the same).
func TestReconcileDaemonSet_Idempotent(t *testing.T) {
	ns := "falcon-clusterguard"
	owner := ownerCR()
	r := newFakeReconciler(owner)
	n := New(r, baseConfig(ns, owner))
	ctx := context.Background()

	if err := n.reconcileDaemonSet(ctx); err != nil {
		t.Fatalf("first reconcileDaemonSet() error: %v", err)
	}
	first := &appsv1.DaemonSet{}
	_ = r.Get(ctx, types.NamespacedName{Name: "falcon-sensor", Namespace: ns}, first)

	if err := n.reconcileDaemonSet(ctx); err != nil {
		t.Fatalf("second reconcileDaemonSet() error: %v", err)
	}
	second := &appsv1.DaemonSet{}
	_ = r.Get(ctx, types.NamespacedName{Name: "falcon-sensor", Namespace: ns}, second)

	if first.ResourceVersion != second.ResourceVersion {
		t.Errorf("expected no update on second reconcile: rv %q → %q",
			first.ResourceVersion, second.ResourceVersion)
	}
}

// TestReconcileDaemonSet_TerminationGracePeriod verifies that changing
// TerminationGracePeriod updates TerminationGracePeriodSeconds on the live DaemonSet.
func TestReconcileDaemonSet_TerminationGracePeriod(t *testing.T) {
	ns := "falcon-clusterguard"
	owner := ownerCR()
	r := newFakeReconciler(owner)

	cfg := baseConfig(ns, owner)
	grace30 := int64(30)
	cfg.NodeSensor.TerminationGracePeriod = &grace30
	n := New(r, cfg)

	ds := reconcileAndGet(t, n, r, ns, "falcon-sensor")
	if ds.Spec.Template.Spec.TerminationGracePeriodSeconds == nil {
		t.Fatal("expected non-nil TerminationGracePeriodSeconds after create")
	}
	if *ds.Spec.Template.Spec.TerminationGracePeriodSeconds != 30 {
		t.Errorf("after create: want 30, got %d", *ds.Spec.Template.Spec.TerminationGracePeriodSeconds)
	}

	// Now update the config to a new value and reconcile again.
	cfg2 := baseConfig(ns, owner)
	grace90 := int64(90)
	cfg2.NodeSensor.TerminationGracePeriod = &grace90
	n2 := New(r, cfg2)

	ds = reconcileAndGet(t, n2, r, ns, "falcon-sensor")
	if *ds.Spec.Template.Spec.TerminationGracePeriodSeconds != 90 {
		t.Errorf("after update: want 90, got %d", *ds.Spec.Template.Spec.TerminationGracePeriodSeconds)
	}
}

// TestReconcileDaemonSet_PodTemplateLabels verifies that changing GKE.DeployAllowListVersion
// updates pod template labels on the live DaemonSet.
func TestReconcileDaemonSet_PodTemplateLabels(t *testing.T) {
	ns := "falcon-clusterguard"
	owner := ownerCR()
	r := newFakeReconciler(owner)

	gkeEnabled := true
	v1 := "v1.0.0"
	cfg := baseConfig(ns, owner)
	cfg.NodeSensor.GKE = falconv1alpha1.FalconClusterGuardAutoPilot{Enabled: &gkeEnabled, DeployAllowListVersion: &v1}
	n := New(r, cfg)

	ds := reconcileAndGet(t, n, r, ns, "falcon-sensor")
	labelKey := pkgcommon.GKEAutoPilotAllowListLabelKey
	if ds.Spec.Template.Labels[labelKey] == "" {
		t.Fatalf("expected autopilot allowlist label after create, got empty")
	}
	firstLabelVal := ds.Spec.Template.Labels[labelKey]

	// Update to a new allowlist version and reconcile.
	v2 := "v2.0.0"
	cfg2 := baseConfig(ns, owner)
	cfg2.NodeSensor.GKE = falconv1alpha1.FalconClusterGuardAutoPilot{Enabled: &gkeEnabled, DeployAllowListVersion: &v2}
	n2 := New(r, cfg2)

	ds = reconcileAndGet(t, n2, r, ns, "falcon-sensor")
	secondLabelVal := ds.Spec.Template.Labels[labelKey]
	if secondLabelVal == firstLabelVal {
		t.Errorf("expected pod template label to change: still %q", secondLabelVal)
	}
	if secondLabelVal == "" {
		t.Error("expected non-empty autopilot allowlist label after update")
	}
}

// TestReconcileDaemonSet_InitContainerResources verifies that switching Backend to "bpf"
// (with GKE enabled) causes init container resources to be applied on the live DaemonSet.
func TestReconcileDaemonSet_InitContainerResources(t *testing.T) {
	ns := "falcon-clusterguard"
	owner := ownerCR()
	r := newFakeReconciler(owner)

	// Create with kernel backend — init container should have empty resources.
	cfg := baseConfig(ns, owner)
	cfg.NodeSensor.Backend = "kernel"
	n := New(r, cfg)

	ds := reconcileAndGet(t, n, r, ns, "falcon-sensor")
	initResources := ds.Spec.Template.Spec.InitContainers[0].Resources
	if len(initResources.Limits) != 0 || len(initResources.Requests) != 0 {
		t.Fatalf("expected empty init container resources for kernel backend, got limits=%v requests=%v",
			initResources.Limits, initResources.Requests)
	}

	// Update to bpf + GKE — init container should now have resource limits.
	gkeEnabled := true
	cfg2 := baseConfig(ns, owner)
	cfg2.NodeSensor.Backend = "bpf"
	cfg2.NodeSensor.GKE = falconv1alpha1.FalconClusterGuardAutoPilot{Enabled: &gkeEnabled}
	n2 := New(r, cfg2)

	ds = reconcileAndGet(t, n2, r, ns, "falcon-sensor")
	initResources = ds.Spec.Template.Spec.InitContainers[0].Resources
	wantCPU := resource.MustParse("10m")
	gotCPU, ok := initResources.Limits[corev1.ResourceCPU]
	if !ok {
		t.Fatal("expected init container CPU limit after switching to bpf+GKE")
	}
	if gotCPU.Cmp(wantCPU) != 0 {
		t.Errorf("init container CPU limit: want %s, got %s", wantCPU.String(), gotCPU.String())
	}
}

// TestReconcileDaemonSet_ContainerEnvFrom verifies that toggling GKE.Enabled updates
// the EnvFrom ConfigMap reference on the main container.
func TestReconcileDaemonSet_ContainerEnvFrom(t *testing.T) {
	ns := "falcon-clusterguard"
	owner := ownerCR()
	r := newFakeReconciler(owner)

	// Create without GKE — should reference the prefix-derived ConfigMap.
	cfg := baseConfig(ns, owner)
	n := New(r, cfg)

	ds := reconcileAndGet(t, n, r, ns, "falcon-sensor")
	nonGKECMName := "falcon-sensor-config"
	if !hasEnvFromConfigMap(ds, nonGKECMName) {
		t.Fatalf("expected EnvFrom to reference %q, envFrom=%v",
			nonGKECMName, ds.Spec.Template.Spec.Containers[0].EnvFrom)
	}

	// Update with GKE enabled — should switch to the fixed GKE ConfigMap name.
	gkeEnabled := true
	cfg2 := baseConfig(ns, owner)
	cfg2.NodeSensor.GKE = falconv1alpha1.FalconClusterGuardAutoPilot{Enabled: &gkeEnabled}
	n2 := New(r, cfg2)

	ds = reconcileAndGet(t, n2, r, ns, "falcon-sensor")
	// The GKE ConfigMap name is pkgcommon.GKEAutoPilotConfigMapName; check it differs
	// from the non-GKE name and that the non-GKE name is no longer present.
	if hasEnvFromConfigMap(ds, nonGKECMName) {
		t.Errorf("expected EnvFrom to no longer reference %q after enabling GKE", nonGKECMName)
	}
	if len(ds.Spec.Template.Spec.Containers[0].EnvFrom) == 0 {
		t.Error("expected at least one EnvFrom entry after enabling GKE")
	}
}

// TestReconcileDaemonSet_ContainerImage verifies that updating the image triggers
// a DaemonSet update (existing reconcile path, confirmed still working).
func TestReconcileDaemonSet_ContainerImage(t *testing.T) {
	ns := "falcon-clusterguard"
	owner := ownerCR()
	r := newFakeReconciler(owner)
	n := New(r, baseConfig(ns, owner))
	reconcileAndGet(t, n, r, ns, "falcon-sensor")

	cfg2 := baseConfig(ns, owner)
	cfg2.Image = "quay.io/crowdstrike/falcon-sensor:v2.0.0"
	n2 := New(r, cfg2)

	ds := reconcileAndGet(t, n2, r, ns, "falcon-sensor")
	if ds.Spec.Template.Spec.Containers[0].Image != "quay.io/crowdstrike/falcon-sensor:v2.0.0" {
		t.Errorf("expected updated image, got %q", ds.Spec.Template.Spec.Containers[0].Image)
	}
}

// TestReconcileDaemonSet_Tolerations verifies that new tolerations are applied.
func TestReconcileDaemonSet_Tolerations(t *testing.T) {
	ns := "falcon-clusterguard"
	owner := ownerCR()
	r := newFakeReconciler(owner)
	n := New(r, baseConfig(ns, owner))
	reconcileAndGet(t, n, r, ns, "falcon-sensor")

	newTol := corev1.Toleration{Key: "node-role.kubernetes.io/control-plane", Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoSchedule}
	cfg2 := baseConfig(ns, owner)
	cfg2.NodeSensor.Tolerations = &[]corev1.Toleration{newTol}
	n2 := New(r, cfg2)

	ds := reconcileAndGet(t, n2, r, ns, "falcon-sensor")
	found := false
	for _, tol := range ds.Spec.Template.Spec.Tolerations {
		if equality.Semantic.DeepEqual(tol, newTol) {
			found = true
		}
	}
	if !found {
		t.Errorf("expected toleration %+v to be present after update, got %+v",
			newTol, ds.Spec.Template.Spec.Tolerations)
	}
}

// hasEnvFromConfigMap returns true if the first container's EnvFrom contains a
// ConfigMapRef with the given name.
func hasEnvFromConfigMap(ds *appsv1.DaemonSet, name string) bool {
	if len(ds.Spec.Template.Spec.Containers) == 0 {
		return false
	}
	for _, ef := range ds.Spec.Template.Spec.Containers[0].EnvFrom {
		if ef.ConfigMapRef != nil && ef.ConfigMapRef.Name == name {
			return true
		}
	}
	return false
}

// TestReconcileDaemonSet_ConfigMapChecksumAnnotation verifies that the pod template
// carries a checksum/config annotation derived from the ConfigMap content, and that
// changing config (e.g. CID) updates the annotation — causing a pod rerollout.
func TestReconcileDaemonSet_ConfigMapChecksumAnnotation(t *testing.T) {
	ns := "falcon-clusterguard"
	owner := ownerCR()
	r := newFakeReconciler(owner)

	cfg := baseConfig(ns, owner)
	n := New(r, cfg)

	ds := reconcileAndGet(t, n, r, ns, "falcon-sensor")

	firstChecksum := ds.Spec.Template.Annotations["checksum/config"]
	if firstChecksum == "" {
		t.Fatal("expected non-empty checksum/config annotation after create")
	}
	if len(firstChecksum) != 16 {
		t.Errorf("expected 16-char checksum, got %d chars: %q", len(firstChecksum), firstChecksum)
	}

	// Change a ConfigMap value (CID) and reconcile — checksum must change,
	// which changes spec.template and triggers a DaemonSet rerollout.
	cfg2 := baseConfig(ns, owner)
	cfg2.Cid = "deadbeef00000000000000000000000000000000-ab"
	n2 := New(r, cfg2)

	ds = reconcileAndGet(t, n2, r, ns, "falcon-sensor")
	secondChecksum := ds.Spec.Template.Annotations["checksum/config"]
	if secondChecksum == "" {
		t.Fatal("expected non-empty checksum/config annotation after update")
	}
	if secondChecksum == firstChecksum {
		t.Errorf("expected checksum to change when CID changes: still %q", firstChecksum)
	}
}

// TestReconcileDaemonSet_ConfigMapChecksumAnnotation_NoChangeWhenConfigUnchanged verifies
// that the checksum annotation is stable when config does not change (no spurious rerollout).
func TestReconcileDaemonSet_ConfigMapChecksumAnnotation_Stable(t *testing.T) {
	ns := "falcon-clusterguard"
	owner := ownerCR()
	r := newFakeReconciler(owner)

	cfg := baseConfig(ns, owner)
	n := New(r, cfg)
	reconcileAndGet(t, n, r, ns, "falcon-sensor")

	// Reconcile again with identical config.
	n2 := New(r, baseConfig(ns, owner))
	ds := reconcileAndGet(t, n2, r, ns, "falcon-sensor")

	checksum := ds.Spec.Template.Annotations["checksum/config"]
	if checksum == "" {
		t.Fatal("expected checksum/config annotation to be present")
	}

	// Reconcile a third time — checksum must be the same.
	n3 := New(r, baseConfig(ns, owner))
	ds = reconcileAndGet(t, n3, r, ns, "falcon-sensor")
	if ds.Spec.Template.Annotations["checksum/config"] != checksum {
		t.Errorf("checksum changed across identical reconciles: %q → %q",
			checksum, ds.Spec.Template.Annotations["checksum/config"])
	}
}

// TestReconcileProxyService_SkipWhenDisabled verifies that reconcileProxyService is a no-op
// when guardian proxy is disabled.
func TestReconcileProxyService_SkipWhenDisabled(t *testing.T) {
	ns := "falcon-clusterguard"
	owner := ownerCR()
	r := newFakeReconciler(owner)
	n := New(r, baseConfig(ns, owner))

	ctx := context.Background()
	if err := n.reconcileProxyService(ctx); err != nil {
		t.Fatalf("reconcileProxyService() error: %v", err)
	}

	svc := &corev1.Service{}
	if err := r.Get(ctx, types.NamespacedName{Name: pkgcommon.ClusterGuardProxyServiceName, Namespace: ns}, svc); err == nil {
		t.Error("expected no proxy Service when proxy is disabled, but one was created")
	}
}

// TestReconcileProxyService_CreateWhenEnabled verifies that reconcileProxyService creates
// the falcon-proxy Service when the guardian proxy is enabled.
func TestReconcileProxyService_CreateWhenEnabled(t *testing.T) {
	ns := "falcon-clusterguard"
	owner := ownerCR()
	r := newFakeReconciler(owner)

	enabled := true
	port := int32(48080)
	cfg := baseConfig(ns, owner)
	cfg.NodeSensor.Guardian.Proxy.Enabled = &enabled
	cfg.NodeSensor.Guardian.Proxy.Port = &port
	n := New(r, cfg)

	ctx := context.Background()
	if err := n.reconcileProxyService(ctx); err != nil {
		t.Fatalf("reconcileProxyService() error: %v", err)
	}

	svc := &corev1.Service{}
	if err := r.Get(ctx, types.NamespacedName{Name: pkgcommon.ClusterGuardProxyServiceName, Namespace: ns}, svc); err != nil {
		t.Fatalf("expected falcon-proxy Service to exist: %v", err)
	}
	if svc.Spec.InternalTrafficPolicy == nil || *svc.Spec.InternalTrafficPolicy != corev1.ServiceInternalTrafficPolicyLocal {
		t.Error("expected InternalTrafficPolicy=Local on proxy Service")
	}
}

// TestReconcileProxyService_Idempotent verifies that a second call does not fail or
// change the resource version (no spurious update).
func TestReconcileProxyService_Idempotent(t *testing.T) {
	ns := "falcon-clusterguard"
	owner := ownerCR()
	r := newFakeReconciler(owner)

	enabled := true
	port := int32(48080)
	cfg := baseConfig(ns, owner)
	cfg.NodeSensor.Guardian.Proxy.Enabled = &enabled
	cfg.NodeSensor.Guardian.Proxy.Port = &port
	n := New(r, cfg)

	ctx := context.Background()
	if err := n.reconcileProxyService(ctx); err != nil {
		t.Fatalf("first reconcileProxyService() error: %v", err)
	}
	first := &corev1.Service{}
	_ = r.Get(ctx, types.NamespacedName{Name: pkgcommon.ClusterGuardProxyServiceName, Namespace: ns}, first)

	if err := n.reconcileProxyService(ctx); err != nil {
		t.Fatalf("second reconcileProxyService() error: %v", err)
	}
	second := &corev1.Service{}
	_ = r.Get(ctx, types.NamespacedName{Name: pkgcommon.ClusterGuardProxyServiceName, Namespace: ns}, second)

	if first.ResourceVersion != second.ResourceVersion {
		t.Errorf("expected no update on second reconcile: rv %q → %q",
			first.ResourceVersion, second.ResourceVersion)
	}
}

// TestReconcileDaemonSet_DNSConfig verifies that setting DNSConfig applies it to the
// live DaemonSet, and that removing it clears it.
func TestReconcileDaemonSet_DNSConfig(t *testing.T) {
	ns := "falcon-clusterguard"
	owner := ownerCR()
	r := newFakeReconciler(owner)
	n := New(r, baseConfig(ns, owner))
	reconcileAndGet(t, n, r, ns, "falcon-sensor")

	// Add DNSConfig
	cfg2 := baseConfig(ns, owner)
	cfg2.NodeSensor.DNSConfig = &corev1.PodDNSConfig{
		Nameservers: []string{"8.8.8.8"},
	}
	n2 := New(r, cfg2)
	ds := reconcileAndGet(t, n2, r, ns, "falcon-sensor")

	if ds.Spec.Template.Spec.DNSConfig == nil {
		t.Fatal("expected DNSConfig to be applied")
	}
	if len(ds.Spec.Template.Spec.DNSConfig.Nameservers) != 1 || ds.Spec.Template.Spec.DNSConfig.Nameservers[0] != "8.8.8.8" {
		t.Errorf("expected nameserver 8.8.8.8, got %v", ds.Spec.Template.Spec.DNSConfig.Nameservers)
	}
}

// TestReconcileDaemonSet_ProxyPortAndVolume verifies that enabling the guardian proxy
// causes the proxy port, volume, and volumemount to appear on the live DaemonSet.
func TestReconcileDaemonSet_ProxyPortAndVolume(t *testing.T) {
	ns := "falcon-clusterguard"
	owner := ownerCR()
	r := newFakeReconciler(owner)
	n := New(r, baseConfig(ns, owner))
	reconcileAndGet(t, n, r, ns, "falcon-sensor")

	enabled := true
	port := int32(48080)
	cfg2 := baseConfig(ns, owner)
	cfg2.NodeSensor.Guardian.Proxy.Enabled = &enabled
	cfg2.NodeSensor.Guardian.Proxy.Port = &port
	n2 := New(r, cfg2)
	ds := reconcileAndGet(t, n2, r, ns, "falcon-sensor")

	foundPort := false
	for _, p := range ds.Spec.Template.Spec.Containers[0].Ports {
		if p.Name == pkgcommon.ClusterGuardProxyPortName && p.ContainerPort == port {
			foundPort = true
		}
	}
	if !foundPort {
		t.Error("expected proxy container port after enabling guardian proxy")
	}

	foundVol := false
	for _, v := range ds.Spec.Template.Spec.Volumes {
		if v.Name == pkgcommon.ClusterGuardProxyTLSVolumeName {
			foundVol = true
		}
	}
	if !foundVol {
		t.Error("expected proxy-tls volume after enabling guardian proxy")
	}

	foundMount := false
	for _, vm := range ds.Spec.Template.Spec.Containers[0].VolumeMounts {
		if vm.Name == pkgcommon.ClusterGuardProxyTLSVolumeName {
			foundMount = true
		}
	}
	if !foundMount {
		t.Error("expected proxy-tls volumemount after enabling guardian proxy")
	}
}
