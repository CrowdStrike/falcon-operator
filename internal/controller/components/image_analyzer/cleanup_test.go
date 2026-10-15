package image_analyzer

import (
	"context"
	"fmt"
	"testing"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/internal/controller/components"
	pkgcommon "github.com/crowdstrike/falcon-operator/pkg/common"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

// controlledBy lists every object of the kinds Image Analyzer creates that is controlled by owner.
func controlledBy(t *testing.T, c client.Client, owner client.Object) []string {
	t.Helper()
	lists := []client.ObjectList{
		&appsv1.DeploymentList{}, &corev1.ServiceList{}, &corev1.SecretList{}, &corev1.ConfigMapList{},
		&corev1.ServiceAccountList{}, &rbacv1.ClusterRoleBindingList{},
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

func TestCleanup_RemovesEverythingReconcileCreated(t *testing.T) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = falconv1alpha1.AddToScheme(scheme)

	owner := ownerCR()
	owner.UID = "uid-test-fcg"
	r := &fakeReconciler{Client: fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(owner).WithStatusSubresource(owner).Build()}
	ia := New(r, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: testNamespace,
			Image:            "quay.io/crowdstrike/falcon-clusterguard:latest",
			Cid:              "abc123-xx",
			Owner:            owner,
			Status:           &owner.Status,
			Request:          ctrl.Request{NamespacedName: types.NamespacedName{Name: owner.Name}},
		},
	})

	if _, err := ia.Reconcile(ctx); err != nil {
		t.Fatalf("Reconcile() error: %v", err)
	}
	if len(controlledBy(t, r, owner)) == 0 {
		t.Fatal("expected Reconcile to create resources controlled by the FalconClusterGuard")
	}
	persisted := &falconv1alpha1.FalconClusterGuard{}
	if err := r.Get(ctx, types.NamespacedName{Name: owner.Name}, persisted); err != nil {
		t.Fatal(err)
	}
	if !meta.IsStatusConditionTrue(persisted.Status.Conditions, falconv1alpha1.ConditionImageAnalyzerReady) {
		t.Fatalf("expected %s condition to be persisted, got %v", falconv1alpha1.ConditionImageAnalyzerReady, persisted.Status.Conditions)
	}

	// The second call verifies Cleanup is idempotent once everything is gone.
	for i := range 2 {
		if err := ia.Cleanup(ctx); err != nil {
			t.Fatalf("call %d: Cleanup() error: %v", i+1, err)
		}
	}

	if left := controlledBy(t, r, owner); len(left) != 0 {
		t.Errorf("Cleanup left resources behind: %v", left)
	}
	if err := r.Get(ctx, types.NamespacedName{Name: owner.Name}, persisted); err != nil {
		t.Fatal(err)
	}
	if c := meta.FindStatusCondition(persisted.Status.Conditions, falconv1alpha1.ConditionImageAnalyzerReady); c != nil {
		t.Errorf("expected %s condition to be removed, got %v", falconv1alpha1.ConditionImageAnalyzerReady, c)
	}
}

func TestCleanup_KeepsResourcesNotControlledByOwner(t *testing.T) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = falconv1alpha1.AddToScheme(scheme)

	owner := ownerCR()
	owner.UID = "uid-test-fcg"
	// Same name the FCG uses, but created by something else (e.g. a standalone FalconImageAnalyzer).
	foreign := &appsv1.Deployment{ObjectMeta: metav1.ObjectMeta{Name: pkgcommon.FCGImageAnalyzerDeploymentName, Namespace: testNamespace}}
	r := &fakeReconciler{Client: fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(owner, foreign).WithStatusSubresource(owner).Build()}
	ia := New(r, Config{BaseConfig: components.BaseConfig{
		InstallNamespace: testNamespace,
		Owner:            owner,
		Status:           &owner.Status,
		Request:          ctrl.Request{NamespacedName: types.NamespacedName{Name: owner.Name}},
	}})

	if err := ia.Cleanup(ctx); err != nil {
		t.Fatalf("Cleanup() error: %v", err)
	}
	if err := r.Get(ctx, client.ObjectKeyFromObject(foreign), &appsv1.Deployment{}); err != nil {
		t.Errorf("expected Deployment not controlled by the FalconClusterGuard to be kept: %v", err)
	}
}
