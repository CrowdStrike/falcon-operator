package controllers

import (
	"testing"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/stretchr/testify/assert"
)

func TestAdmissionDisabledNamespaces(t *testing.T) {
	tests := []struct {
		name             string
		installNamespace string
		userNamespaces   []string
		want             []string
	}{
		{
			name:             "default install namespace",
			installNamespace: "falcon-kac",
			want:             []string{"falcon-kac", "kube-system", "kube-public", "falcon-system", "falcon-kubernetes-protection"},
		},
		{
			name:             "custom install namespace and user namespaces",
			installNamespace: "my-kac",
			userNamespaces:   []string{"team-a"},
			want:             []string{"my-kac", "kube-system", "kube-public", "falcon-system", "falcon-kubernetes-protection", "team-a"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			falconAdmission := &falconv1alpha1.FalconAdmission{}
			falconAdmission.Spec.InstallNamespace = tt.installNamespace
			falconAdmission.Spec.AdmissionConfig.DisabledNamespaces.Namespaces = tt.userNamespaces

			assert.Equal(t, tt.want, admissionDisabledNamespaces(falconAdmission))
		})
	}
}
