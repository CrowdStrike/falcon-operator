package controllers

import (
	"context"
	"testing"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/crowdstrike/falcon-operator/pkg/tls"
	"github.com/go-logr/logr"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestReconcileAPITLSSecretsKeepsPKIConsistent(t *testing.T) {
	ctx := context.Background()
	scheme := runtime.NewScheme()
	_ = clientgoscheme.AddToScheme(scheme)
	_ = falconv1alpha1.AddToScheme(scheme)
	const namespace = "falcon-clusterguard"

	fcg := &falconv1alpha1.FalconClusterGuard{
		ObjectMeta: metav1.ObjectMeta{Name: "test-fcg", UID: "uid-test-fcg"},
		Spec:       falconv1alpha1.FalconClusterGuardSpec{InstallNamespace: namespace},
	}
	c := fake.NewClientBuilder().WithScheme(scheme).WithObjects(fcg).WithStatusSubresource(fcg).Build()
	r := &FalconClusterGuardReconciler{Client: c, Reader: c, RuntimeScheme: scheme, log: logr.Discard()}
	req := ctrl.Request{NamespacedName: types.NamespacedName{Name: fcg.Name}}

	get := func(name string) *corev1.Secret {
		t.Helper()
		s := &corev1.Secret{}
		if err := c.Get(ctx, types.NamespacedName{Name: name, Namespace: namespace}, s); err != nil {
			t.Fatalf("get Secret %s: %v", name, err)
		}
		return s
	}
	reconcile := func() []byte {
		t.Helper()
		if err := r.reconcileAPITLSSecrets(ctx, req, fcg); err != nil {
			t.Fatalf("reconcileAPITLSSecrets() error: %v", err)
		}
		ca := get(common.ClusterGuardControllerAPICASecretName).Data["ca.crt"]
		for _, name := range []string{common.ClusterGuardControllerAPITLSSecretName, common.ClusterGuardNodeSensorTLSSecretName} {
			if err := tls.FCGCheckSignedBy(get(name).Data["tls.crt"], ca); err != nil {
				t.Fatalf("%s is not signed by the CA: %v", name, err)
			}
		}
		return ca
	}

	ca := reconcile()

	if got := reconcile(); string(got) != string(ca) {
		t.Error("a consistent set of secrets was regenerated")
	}

	if err := c.Delete(ctx, get(common.ClusterGuardNodeSensorTLSSecretName)); err != nil {
		t.Fatal(err)
	}
	got := reconcile()
	if string(got) == string(ca) {
		t.Error("a missing leaf secret did not regenerate the CA")
	}
	ca = got

	otherCA, _, err := tls.FCGGenerateCA("other-ca", 1)
	if err != nil {
		t.Fatal(err)
	}
	caSecret := get(common.ClusterGuardControllerAPICASecretName)
	caSecret.Data["ca.crt"] = otherCA
	if err := c.Update(ctx, caSecret); err != nil {
		t.Fatal(err)
	}
	if got := reconcile(); string(got) == string(ca) || string(got) == string(otherCA) {
		t.Error("a CA that did not sign the leaves was not regenerated")
	}
}
