package assets

import (
	"testing"

	"github.com/crowdstrike/falcon-operator/pkg/common"
	"github.com/google/go-cmp/cmp"
	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TestServiceAccount tests the ServiceAccount function
func TestServiceAccount(t *testing.T) {
	name := "test"
	namespace := "test"
	component := common.FalconAdmissionController
	annotations := map[string]string{}
	imagePullSecrets := []corev1.LocalObjectReference{}
	want := &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "ServiceAccount",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      common.CRLabels("serviceaccount", name, component),
			Annotations: annotations,
		},
		ImagePullSecrets: imagePullSecrets,
	}
	got := ServiceAccount(name, namespace, component, annotations, imagePullSecrets)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("ServiceAccount() mismatch (-want +got): %s", diff)
	}
}

// TestClusterRoleBinding tests the ClusterRoleBinding function
func TestClusterRoleBinding(t *testing.T) {
	name := "test"
	namespace := "test"
	clusterrole := "test"
	sa := "test"
	component := common.FalconAdmissionController
	subj := []rbacv1.Subject{}
	want := &rbacv1.ClusterRoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "ClusterRoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: common.CRLabels("clusterrolebinding", name, component),
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      sa,
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     clusterrole,
		},
	}
	got := ClusterRoleBinding(name, namespace, clusterrole, sa, component, subj)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("ClusterRoleBinding() mismatch (-want +got): %s", diff)
	}
}

// TestRole tests the Role function
func TestRole(t *testing.T) {
	name := "test"
	namespace := "test"
	want := &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "Role",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels:    common.CRLabels("role", name, common.FalconAdmissionController),
		},
	}
	got := Role(name, namespace)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("Role() mismatch (-want +got): %s", diff)
	}
}

// TestRoleBinding tests the RoleBinding function
func TestRoleBinding(t *testing.T) {
	name := "test"
	namespace := "test"
	role := "test"
	sa := "test"
	component := common.FalconAdmissionController

	want := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rbacv1.SchemeGroupVersion.String(),
			Kind:       "RoleBinding",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:   name,
			Labels: common.CRLabels("rolebinding", name, component),
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      sa,
				Namespace: namespace,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     role,
		},
	}
	got := RoleBinding(name, namespace, role, sa)
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("RoleBinding() mismatch (-want +got): %s", diff)
	}
}
