package clusterguard_controller

import (
	"context"
	"fmt"
	"testing"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/internal/controller/components"
	pkgcommon "github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/go-logr/logr"
	arv1 "k8s.io/api/admissionregistration/v1"
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

const cleanupTestNamespace = "falcon-clusterguard"

type fakeReconciler struct {
	client.Client
}

func (f *fakeReconciler) GetK8sReader() client.Reader { return f.Client }
func (f *fakeReconciler) GetScheme() *runtime.Scheme  { return f.Client.Scheme() }
func (f *fakeReconciler) GetLog() logr.Logger         { return logr.Discard() }

func newCleanupTestController(t *testing.T, objs ...client.Object) (*fakeReconciler, *falconv1alpha1.FalconClusterGuard, *ClusterGuardController) {
	t.Helper()
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = falconv1alpha1.AddToScheme(scheme)

	owner := &falconv1alpha1.FalconClusterGuard{
		TypeMeta:   metav1.TypeMeta{APIVersion: falconv1alpha1.GroupVersion.String(), Kind: "FalconClusterGuard"},
		ObjectMeta: metav1.ObjectMeta{Name: "test-fcg", UID: "uid-test-fcg"},
	}
	r := &fakeReconciler{Client: fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(append(objs, owner)...).WithStatusSubresource(owner).Build()}
	clusterName := "test-cluster"
	return r, owner, New(r, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: cleanupTestNamespace,
			Image:            "quay.io/crowdstrike/falcon-clusterguard:latest",
			Cid:              "abc123-xx",
			Owner:            owner,
			Status:           &owner.Status,
			Request:          ctrl.Request{NamespacedName: types.NamespacedName{Name: owner.Name}},
		},
		ClusterName: &clusterName,
	})
}

// controlledBy lists every object of the kinds the admission controller creates that is controlled by owner.
func controlledBy(t *testing.T, c client.Client, owner client.Object) []string {
	t.Helper()
	lists := []client.ObjectList{
		&arv1.ValidatingWebhookConfigurationList{}, &appsv1.DeploymentList{}, &corev1.ServiceList{},
		&corev1.SecretList{}, &corev1.ConfigMapList{}, &corev1.ResourceQuotaList{}, &rbacv1.RoleBindingList{},
		&rbacv1.RoleList{}, &rbacv1.ClusterRoleBindingList{}, &corev1.ServiceAccountList{},
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
	r, owner, a := newCleanupTestController(t)

	if _, err := a.Reconcile(ctx); err != nil {
		t.Fatalf("Reconcile() error: %v", err)
	}
	if len(controlledBy(t, r, owner)) == 0 {
		t.Fatal("expected Reconcile to create resources controlled by the FalconClusterGuard")
	}
	persisted := &falconv1alpha1.FalconClusterGuard{}
	if err := r.Get(ctx, types.NamespacedName{Name: owner.Name}, persisted); err != nil {
		t.Fatal(err)
	}
	if !meta.IsStatusConditionTrue(persisted.Status.Conditions, falconv1alpha1.ConditionAdmissionReady) {
		t.Fatalf("expected %s condition to be persisted, got %v", falconv1alpha1.ConditionAdmissionReady, persisted.Status.Conditions)
	}

	// The second call verifies Cleanup is idempotent once everything is gone.
	for i := range 2 {
		if err := a.Cleanup(ctx); err != nil {
			t.Fatalf("call %d: Cleanup() error: %v", i+1, err)
		}
	}

	if left := controlledBy(t, r, owner); len(left) != 0 {
		t.Errorf("Cleanup left resources behind: %v", left)
	}
	if err := r.Get(ctx, types.NamespacedName{Name: owner.Name}, persisted); err != nil {
		t.Fatal(err)
	}
	if c := meta.FindStatusCondition(persisted.Status.Conditions, falconv1alpha1.ConditionAdmissionReady); c != nil {
		t.Errorf("expected %s condition to be removed, got %v", falconv1alpha1.ConditionAdmissionReady, c)
	}
}

func TestCleanup_KeepsSharedResources(t *testing.T) {
	ctx := context.Background()
	// falcon-kac-meta is also written by a standalone FalconAdmission; the API TLS secret is shared with the node sensor.
	standaloneMeta := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{
		Name: pkgcommon.FalconAdmissionClusterNameConfigMapName, Namespace: cleanupTestNamespace}}
	apiTLS := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{
		Name: pkgcommon.ClusterGuardControllerAPITLSSecretName, Namespace: cleanupTestNamespace}}
	r, owner, a := newCleanupTestController(t, standaloneMeta)
	if err := ctrl.SetControllerReference(owner, apiTLS, r.Scheme()); err != nil {
		t.Fatal(err)
	}
	if err := r.Create(ctx, apiTLS); err != nil {
		t.Fatal(err)
	}

	if err := a.Cleanup(ctx); err != nil {
		t.Fatalf("Cleanup() error: %v", err)
	}
	if err := r.Get(ctx, client.ObjectKeyFromObject(standaloneMeta), &corev1.ConfigMap{}); err != nil {
		t.Errorf("expected ConfigMap not controlled by the FalconClusterGuard to be kept: %v", err)
	}
	if err := r.Get(ctx, client.ObjectKeyFromObject(apiTLS), &corev1.Secret{}); err != nil {
		t.Errorf("expected shared API TLS secret to be kept: %v", err)
	}
}
