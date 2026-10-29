package clusterguard_controller

import (
	"testing"

	"github.com/crowdstrike/falcon-operator/internal/controller/components"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
)

// The falcon-clusterguard Helm chart puts its cgControllerLabels on every cg-controller resource.
func TestClusterGuardControllerResourcesHaveHelmAppLabels(t *testing.T) {
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			Request:          ctrl.Request{NamespacedName: types.NamespacedName{Name: "my-fcg"}},
			InstallNamespace: "falcon-clusterguard",
		},
		ClusterName: ptr.To("my-cluster"),
	})

	want := map[string]string{
		common.KubernetesNameKey:      "falcon-clusterguard",
		common.KubernetesInstanceKey:  "my-fcg",
		common.KubernetesManagedByKey: "falcon-operator",
		common.KubernetesComponentKey: "fcg-controller",
	}

	dep := a.Deployment()
	objs := map[string]map[string]string{
		"Deployment":                     dep.Labels,
		"Deployment pod template":        dep.Spec.Template.Labels,
		"ServiceAccount":                 a.serviceAccount().Labels,
		"ClusterRoleBinding":             a.clusterRoleBinding().Labels,
		"RoleBinding":                    a.roleBinding().Labels,
		"Role":                           a.role().Labels,
		"ResourceQuota":                  a.resourceQuota().Labels,
		"ConfigMap":                      a.configMap().Labels,
		"cluster name ConfigMap":         a.clusterNameConfigMap().Labels,
		"webhook Service":                a.webhookService().Labels,
		"API Service":                    a.apiService().Labels,
		"ValidatingWebhookConfiguration": a.ValidatingWebhook(nil).Labels,
	}
	for obj, got := range objs {
		for k, v := range want {
			if got[k] != v {
				t.Errorf("%s: expected label %s=%q, got %q", obj, k, v, got[k])
			}
		}
	}

	// Pod labels are added on top of the selector; the selector itself must stay unchanged.
	if _, ok := dep.Spec.Selector.MatchLabels[common.KubernetesInstanceKey]; ok {
		t.Errorf("Deployment selector must not include %s: %v", common.KubernetesInstanceKey, dep.Spec.Selector.MatchLabels)
	}
}
