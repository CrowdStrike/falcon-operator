package image_analyzer

import (
	"testing"

	"github.com/crowdstrike/falcon-operator/internal/controller/components"
	"k8s.io/apimachinery/pkg/labels"
)

func TestSelectorsMatchImageAnalyzerPods(t *testing.T) {
	ia := New(nil, Config{BaseConfig: components.BaseConfig{InstallNamespace: "falcon-clusterguard"}})
	dep := ia.deployment()
	podLabels := labels.Set(dep.Spec.Template.Labels)

	selectors := map[string]map[string]string{
		"Deployment": dep.Spec.Selector.MatchLabels,
		"Service":    ia.service().Spec.Selector,
	}
	for name, s := range selectors {
		if sel := labels.SelectorFromSet(s); !sel.Matches(podLabels) {
			t.Errorf("%s selector %v does not match image analyzer pod labels %v", name, sel, podLabels)
		}
	}
}
