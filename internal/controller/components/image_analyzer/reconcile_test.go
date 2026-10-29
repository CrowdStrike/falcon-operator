package image_analyzer

import (
	"context"
	"testing"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/internal/controller/components"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// TestReconcile_ReportsDeploymentReadiness checks that ImageAnalyzerReady follows the Deployment rollout.
func TestReconcile_ReportsDeploymentReadiness(t *testing.T) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = falconv1alpha1.AddToScheme(scheme)

	owner := ownerCR()
	r := &fakeReconciler{Client: fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(owner).WithStatusSubresource(owner).Build()}
	ia := New(r, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: testNamespace,
			Image:            "quay.io/crowdstrike/falcon-clusterguard:latest",
			Owner:            owner,
			Status:           &owner.Status,
			Request:          ctrl.Request{NamespacedName: types.NamespacedName{Name: owner.Name}},
		},
	})
	reconcileCondition := func() *metav1.Condition {
		t.Helper()
		if _, err := ia.Reconcile(ctx); err != nil {
			t.Fatalf("Reconcile() error: %v", err)
		}
		persisted := &falconv1alpha1.FalconClusterGuard{}
		if err := r.Get(ctx, types.NamespacedName{Name: owner.Name}, persisted); err != nil {
			t.Fatal(err)
		}
		c := meta.FindStatusCondition(persisted.Status.Conditions, falconv1alpha1.ConditionImageAnalyzerReady)
		if c == nil {
			t.Fatalf("%s condition not set", falconv1alpha1.ConditionImageAnalyzerReady)
		}
		return c
	}

	if c := reconcileCondition(); c.Status != metav1.ConditionFalse || c.Reason != falconv1alpha1.ReasonProgressing {
		t.Errorf("before rollout: got %s/%s, want False/%s", c.Status, c.Reason, falconv1alpha1.ReasonProgressing)
	}

	dep := getDeployment(t, r)
	replicas := int32(1)
	if dep.Spec.Replicas != nil {
		replicas = *dep.Spec.Replicas
	}
	dep.Status = appsv1.DeploymentStatus{ObservedGeneration: dep.Generation, Replicas: replicas, UpdatedReplicas: replicas, AvailableReplicas: replicas}
	if err := r.Status().Update(ctx, dep); err != nil {
		t.Fatal(err)
	}
	if c := reconcileCondition(); c.Status != metav1.ConditionTrue {
		t.Errorf("after rollout: got %s/%s %q, want True", c.Status, c.Reason, c.Message)
	}
}
