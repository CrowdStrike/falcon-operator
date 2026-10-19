package image_analyzer

import (
	"context"
	"testing"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/internal/controller/components"
	pkgcommon "github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

const testNamespace = "falcon-clusterguard"

// fakeReconciler satisfies the Reconciler interface using a fake client.
type fakeReconciler struct {
	client.Client
}

func (f *fakeReconciler) GetK8sReader() client.Reader { return f.Client }
func (f *fakeReconciler) GetScheme() *runtime.Scheme  { return f.Client.Scheme() }
func (f *fakeReconciler) GetLog() logr.Logger         { return logr.Discard() }

func newFakeReconciler(owner *falconv1alpha1.FalconClusterGuard) *fakeReconciler {
	scheme := runtime.NewScheme()
	_ = corev1.AddToScheme(scheme)
	_ = appsv1.AddToScheme(scheme)
	_ = falconv1alpha1.AddToScheme(scheme)

	// WithStatusSubresource is required for ConditionsUpdate to call Status().Update().
	return &fakeReconciler{
		Client: fake.NewClientBuilder().
			WithScheme(scheme).
			WithObjects(owner).
			WithStatusSubresource(owner).
			Build(),
	}
}

// ownerCR returns a minimal cluster-scoped FalconClusterGuard CR for use as the reconcile owner.
func ownerCR() *falconv1alpha1.FalconClusterGuard {
	return &falconv1alpha1.FalconClusterGuard{
		TypeMeta: metav1.TypeMeta{
			APIVersion: falconv1alpha1.GroupVersion.String(),
			Kind:       "FalconClusterGuard",
		},
		ObjectMeta: metav1.ObjectMeta{Name: "test-fcg"},
	}
}

func newTestIA(r *fakeReconciler, owner *falconv1alpha1.FalconClusterGuard, spec falconv1alpha1.FalconClusterGuardImageAnalyzerSpec) *ImageAnalyzer {
	return New(r, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: testNamespace,
			Image:            "quay.io/crowdstrike/falcon-clusterguard:latest",
			Owner:            owner,
			Status:           &falconv1alpha1.FalconCRStatus{},
			// Request must match the owner so ConditionsUpdate can re-fetch it.
			Request: ctrl.Request{NamespacedName: types.NamespacedName{Name: owner.Name}},
		},
		ImageAnalyzerSpec: spec,
	})
}

func reconcileAndGetDeployment(t *testing.T, ia *ImageAnalyzer, r *fakeReconciler) *appsv1.Deployment {
	t.Helper()
	ctx := context.Background()
	if err := ia.reconcileDeployment(ctx); err != nil {
		t.Fatalf("reconcileDeployment() error: %v", err)
	}
	return getDeployment(t, r)
}

func getDeployment(t *testing.T, r *fakeReconciler) *appsv1.Deployment {
	t.Helper()
	got := &appsv1.Deployment{}
	key := types.NamespacedName{Name: pkgcommon.FCGImageAnalyzerDeploymentName, Namespace: testNamespace}
	if err := r.Get(context.Background(), key, got); err != nil {
		t.Fatalf("Get Deployment: %v", err)
	}
	return got
}

func envValue(env []corev1.EnvVar, name string) (string, bool) {
	for _, e := range env {
		if e.Name == name {
			return e.Value, true
		}
	}
	return "", false
}

func TestReconcileDeployment_Create(t *testing.T) {
	owner := ownerCR()
	r := newFakeReconciler(owner)
	ia := newTestIA(r, owner, falconv1alpha1.FalconClusterGuardImageAnalyzerSpec{})

	dep := reconcileAndGetDeployment(t, ia, r)

	if got := dep.Spec.Template.Spec.Containers[0].Image; got != "quay.io/crowdstrike/falcon-clusterguard:latest" {
		t.Errorf("expected image %q, got %q", "quay.io/crowdstrike/falcon-clusterguard:latest", got)
	}
}

func TestReconcileDeployment_Idempotent(t *testing.T) {
	owner := ownerCR()
	r := newFakeReconciler(owner)
	ia := newTestIA(r, owner, falconv1alpha1.FalconClusterGuardImageAnalyzerSpec{
		Tolerations: []corev1.Toleration{{Key: "dedicated", Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoSchedule}},
	})

	first := reconcileAndGetDeployment(t, ia, r)
	second := reconcileAndGetDeployment(t, ia, r)

	if first.ResourceVersion != second.ResourceVersion {
		t.Errorf("expected no update on second reconcile: rv %q -> %q", first.ResourceVersion, second.ResourceVersion)
	}
}

func TestReconcileDeployment_ImageUpdated(t *testing.T) {
	owner := ownerCR()
	r := newFakeReconciler(owner)
	ia := newTestIA(r, owner, falconv1alpha1.FalconClusterGuardImageAnalyzerSpec{})
	reconcileAndGetDeployment(t, ia, r)

	ia.cfg.Image = "my-registry/falcon-clusterguard:2.0"
	dep := reconcileAndGetDeployment(t, ia, r)

	if got := dep.Spec.Template.Spec.Containers[0].Image; got != "my-registry/falcon-clusterguard:2.0" {
		t.Errorf("expected updated image, got %q", got)
	}
}

func TestReconcileDeployment_PortsRestored(t *testing.T) {
	owner := ownerCR()
	r := newFakeReconciler(owner)
	ia := newTestIA(r, owner, falconv1alpha1.FalconClusterGuardImageAnalyzerSpec{})
	want := reconcileAndGetDeployment(t, ia, r).Spec.Template.Spec.Containers[0].Ports

	drifted := getDeployment(t, r)
	drifted.Spec.Template.Spec.Containers[0].Ports = []corev1.ContainerPort{{Name: "other", ContainerPort: 9999, Protocol: corev1.ProtocolTCP}}
	if err := r.Update(context.Background(), drifted); err != nil {
		t.Fatalf("Update drifted Deployment: %v", err)
	}

	got := reconcileAndGetDeployment(t, ia, r).Spec.Template.Spec.Containers[0].Ports
	if len(got) != len(want) || got[0].ContainerPort != want[0].ContainerPort {
		t.Errorf("expected ports %v to be restored, got %v", want, got)
	}
}

func TestReconcileDeployment_PreservesExternalTolerations(t *testing.T) {
	owner := ownerCR()
	r := newFakeReconciler(owner)
	specTol := corev1.Toleration{Key: "dedicated", Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoSchedule}
	ia := newTestIA(r, owner, falconv1alpha1.FalconClusterGuardImageAnalyzerSpec{Tolerations: []corev1.Toleration{specTol}})
	reconcileAndGetDeployment(t, ia, r)

	externalTol := corev1.Toleration{Key: "platform.example.com/injected", Operator: corev1.TolerationOpExists, Effect: corev1.TaintEffectNoExecute}
	existing := getDeployment(t, r)
	existing.Spec.Template.Spec.Tolerations = append(existing.Spec.Template.Spec.Tolerations, externalTol)
	if err := r.Update(context.Background(), existing); err != nil {
		t.Fatalf("Update Deployment: %v", err)
	}

	tols := reconcileAndGetDeployment(t, ia, r).Spec.Template.Spec.Tolerations
	var hasSpec, hasExternal bool
	for _, tol := range tols {
		switch tol.Key {
		case specTol.Key:
			hasSpec = true
		case externalTol.Key:
			hasExternal = true
		}
	}
	if !hasSpec || !hasExternal {
		t.Errorf("expected spec and external tolerations to be kept, got %v", tols)
	}
}

func TestReconcileDeployment_NodeAffinityOnExistingWithoutAffinity(t *testing.T) {
	owner := ownerCR()
	r := newFakeReconciler(owner)
	ia := newTestIA(r, owner, falconv1alpha1.FalconClusterGuardImageAnalyzerSpec{})
	reconcileAndGetDeployment(t, ia, r)

	existing := getDeployment(t, r)
	existing.Spec.Template.Spec.Affinity = nil
	if err := r.Update(context.Background(), existing); err != nil {
		t.Fatalf("Update Deployment: %v", err)
	}

	dep := reconcileAndGetDeployment(t, ia, r)
	if dep.Spec.Template.Spec.Affinity == nil || dep.Spec.Template.Spec.Affinity.NodeAffinity == nil {
		t.Errorf("expected NodeAffinity to be restored, got %v", dep.Spec.Template.Spec.Affinity)
	}
}

func TestReconcileDeployment_ProxyEnv(t *testing.T) {
	t.Setenv("HTTPS_PROXY", "http://proxy.example.com:3128")

	owner := ownerCR()
	r := newFakeReconciler(owner)
	ia := newTestIA(r, owner, falconv1alpha1.FalconClusterGuardImageAnalyzerSpec{})

	env := reconcileAndGetDeployment(t, ia, r).Spec.Template.Spec.Containers[0].Env
	if v, ok := envValue(env, "HTTPS_PROXY"); !ok || v != "http://proxy.example.com:3128" {
		t.Fatalf("expected HTTPS_PROXY on create, got %q (present=%v)", v, ok)
	}

	t.Setenv("HTTPS_PROXY", "http://proxy2.example.com:3128")
	env = reconcileAndGetDeployment(t, ia, r).Spec.Template.Spec.Containers[0].Env
	if v, _ := envValue(env, "HTTPS_PROXY"); v != "http://proxy2.example.com:3128" {
		t.Errorf("expected HTTPS_PROXY to be updated, got %q", v)
	}
}

func TestTriggerRollingRestart_BumpsConfigVersion(t *testing.T) {
	owner := ownerCR()
	r := newFakeReconciler(owner)
	ia := newTestIA(r, owner, falconv1alpha1.FalconClusterGuardImageAnalyzerSpec{})
	reconcileAndGetDeployment(t, ia, r)
	ctx := context.Background()

	for _, want := range []string{"1", "2"} {
		if err := ia.triggerRollingRestart(ctx); err != nil {
			t.Fatalf("triggerRollingRestart() error: %v", err)
		}
		if got := getDeployment(t, r).Spec.Template.Annotations["falcon.config.version"]; got != want {
			t.Errorf("expected falcon.config.version=%q, got %q", want, got)
		}
	}
}
