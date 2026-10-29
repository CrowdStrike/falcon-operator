package image_analyzer

import (
	"testing"

	"github.com/crowdstrike/falcon-operator/internal/controller/components"
	pkgcommon "github.com/crowdstrike/falcon-operator/pkg/common"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

// The falcon-clusterguard Helm chart puts its iarLabels on every image analyzer resource.
func TestImageAnalyzerResourcesHaveHelmAppLabels(t *testing.T) {
	ia := New(nil, Config{
		BaseConfig: components.BaseConfig{
			Request:          ctrl.Request{NamespacedName: types.NamespacedName{Name: "my-fcg"}},
			InstallNamespace: "falcon-clusterguard",
		},
	})

	want := map[string]string{
		pkgcommon.KubernetesNameKey:      "falcon-clusterguard-iar",
		pkgcommon.KubernetesInstanceKey:  "my-fcg",
		pkgcommon.KubernetesManagedByKey: "falcon-operator",
		pkgcommon.KubernetesComponentKey: "iar",
	}

	dep := ia.deployment()
	cm, err := ia.buildConfigMap()
	if err != nil {
		t.Fatalf("buildConfigMap returned error: %v", err)
	}
	objs := map[string]map[string]string{
		"Deployment":              dep.Labels,
		"Deployment pod template": dep.Spec.Template.Labels,
		"ConfigMap":               cm.Labels,
	}
	for obj, got := range objs {
		for k, v := range want {
			if got[k] != v {
				t.Errorf("%s: expected label %s=%q, got %q", obj, k, v, got[k])
			}
		}
	}

	// The Service keeps the standalone FalconImageAnalyzer name for clusterguard controller -> IAR communication.
	svc := ia.service().Labels
	want[pkgcommon.KubernetesNameKey] = pkgcommon.FCGImageAnalyzerDeploymentName
	for k, v := range want {
		if svc[k] != v {
			t.Errorf("Service: expected label %s=%q, got %q", k, v, svc[k])
		}
	}

	if _, ok := dep.Spec.Selector.MatchLabels[pkgcommon.KubernetesInstanceKey]; ok {
		t.Errorf("Deployment selector must not include %s: %v", pkgcommon.KubernetesInstanceKey, dep.Spec.Selector.MatchLabels)
	}
}
