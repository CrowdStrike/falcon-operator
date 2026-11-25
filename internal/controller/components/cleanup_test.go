package components

import (
	"context"
	"testing"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

type fakeReconciler struct {
	client.Client
}

func (f *fakeReconciler) GetK8sReader() client.Reader { return f.Client }
func (f *fakeReconciler) GetScheme() *runtime.Scheme  { return f.Client.Scheme() }
func (f *fakeReconciler) GetLog() logr.Logger         { return logr.Discard() }

func newScheme(t *testing.T) *runtime.Scheme {
	t.Helper()
	scheme := runtime.NewScheme()
	if err := clientgoscheme.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	if err := falconv1alpha1.AddToScheme(scheme); err != nil {
		t.Fatal(err)
	}
	return scheme
}

func owner(name string) *falconv1alpha1.FalconClusterGuard {
	return &falconv1alpha1.FalconClusterGuard{
		TypeMeta:   metav1.TypeMeta{APIVersion: falconv1alpha1.GroupVersion.String(), Kind: "FalconClusterGuard"},
		ObjectMeta: metav1.ObjectMeta{Name: name, UID: types.UID("uid-" + name)},
	}
}

func ownedBy(t *testing.T, scheme *runtime.Scheme, o *falconv1alpha1.FalconClusterGuard, obj client.Object) client.Object {
	t.Helper()
	if err := ctrl.SetControllerReference(o, obj, scheme); err != nil {
		t.Fatal(err)
	}
	return obj
}

func exists(t *testing.T, r client.Client, obj client.Object) bool {
	t.Helper()
	err := r.Get(context.Background(), client.ObjectKeyFromObject(obj), obj)
	if apierrors.IsNotFound(err) {
		return false
	}
	if err != nil {
		t.Fatal(err)
	}
	return true
}

func TestDeleteOwned(t *testing.T) {
	scheme := newScheme(t)
	fcg := owner("fcg")
	other := owner("other")

	owned := ownedBy(t, scheme, fcg, &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "owned", Namespace: "ns"}})
	ownedCluster := ownedBy(t, scheme, fcg, &rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "owned-crb"}})
	foreign := ownedBy(t, scheme, other, &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "foreign", Namespace: "ns"}})
	unowned := &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "unowned", Namespace: "ns"}}

	r := &fakeReconciler{Client: fake.NewClientBuilder().WithScheme(scheme).
		WithObjects(owned, ownedCluster, foreign, unowned).Build()}

	refs := func() []client.Object {
		return []client.Object{
			&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "owned", Namespace: "ns"}},
			&rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "owned-crb"}},
			&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "foreign", Namespace: "ns"}},
			&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "unowned", Namespace: "ns"}},
			&corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "missing", Namespace: "ns"}},
		}
	}

	// A second call must be a no-op now that the owned objects are gone.
	for i := range 2 {
		if err := DeleteOwned(context.Background(), r, fcg, refs()...); err != nil {
			t.Fatalf("call %d: DeleteOwned() error: %v", i+1, err)
		}
	}

	if exists(t, r, &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "owned", Namespace: "ns"}}) {
		t.Error("owned ConfigMap should be deleted")
	}
	if exists(t, r, &rbacv1.ClusterRoleBinding{ObjectMeta: metav1.ObjectMeta{Name: "owned-crb"}}) {
		t.Error("owned ClusterRoleBinding should be deleted")
	}
	if !exists(t, r, &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "foreign", Namespace: "ns"}}) {
		t.Error("ConfigMap controlled by another owner must not be deleted")
	}
	if !exists(t, r, &corev1.ConfigMap{ObjectMeta: metav1.ObjectMeta{Name: "unowned", Namespace: "ns"}}) {
		t.Error("ConfigMap without an owner must not be deleted")
	}
}
