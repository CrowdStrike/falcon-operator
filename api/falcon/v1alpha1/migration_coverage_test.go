package v1alpha1_test

// TestMigrationScriptFieldCoverage ensures that every JSON field on the structs
// used by the FalconDeployment → FalconClusterGuard migration script is
// accounted for in the known-fields list below.
//
// When you add a field to any of these structs, this test will fail. Update the
// knownFields map AND hack/migrate_to_clusterguard.py before marking it green.

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
)

func jsonFieldNames(t reflect.Type) []string {
	var names []string
	for field := range t.Fields() {
		tag := field.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		name, _, _ := strings.Cut(tag, ",")
		if name != "" && name != "-" {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func assertFieldsCovered(t *testing.T, structName string, typ reflect.Type, known []string) {
	t.Helper()
	actual := jsonFieldNames(typ)
	knownSet := make(map[string]bool, len(known))
	for _, f := range known {
		knownSet[f] = true
	}

	var uncovered []string
	for _, f := range actual {
		if !knownSet[f] {
			uncovered = append(uncovered, f)
		}
	}
	if len(uncovered) > 0 {
		t.Errorf(
			"%s has new field(s) not covered by the migration script: %v\n"+
				"Update hack/migrate_to_clusterguard.py and the knownFields list in this test.",
			structName, uncovered,
		)
	}

	var removed []string
	actualSet := make(map[string]bool, len(actual))
	for _, f := range actual {
		actualSet[f] = true
	}
	for _, f := range known {
		if !actualSet[f] {
			removed = append(removed, f)
		}
	}
	if len(removed) > 0 {
		t.Errorf(
			"%s: field(s) in the known list no longer exist on the struct: %v\n"+
				"Remove them from the knownFields list in this test.",
			structName, removed,
		)
	}
}

func TestMigrationScriptFieldCoverage(t *testing.T) {
	tests := []struct {
		name   string
		typ    reflect.Type
		fields []string
	}{
		{
			name: "FalconNodeSensorSpec",
			typ:  reflect.TypeFor[falconv1alpha1.FalconNodeSensorSpec](),
			fields: []string{
				"falcon",
				"falcon_api",
				"falconSecret",
				"installNamespace",
				"internal",
				"node",
			},
		},
		{
			name: "FalconNodeSensorConfig",
			typ:  reflect.TypeFor[falconv1alpha1.FalconNodeSensorConfig](),
			fields: []string{
				"advanced",
				"backend",
				"clusterName",
				"disableCleanup",
				"gke",
				"image",
				"imagePullPolicy",
				"imagePullSecrets",
				"nodeAffinity",
				"priorityClass",
				"resources",
				"serviceAccount",
				"terminationGracePeriod",
				"tolerations",
				"updateStrategy",
				"version",
			},
		},
		{
			name: "FalconAdmissionSpec",
			typ:  reflect.TypeFor[falconv1alpha1.FalconAdmissionSpec](),
			fields: []string{
				"admissionConfig",
				"clusterName",
				"falcon",
				"falcon_api",
				"falconSecret",
				"image",
				"installNamespace",
				"registry",
				"resourcequota",
				"version",
			},
		},
		{
			name: "FalconAdmissionConfigSpec",
			typ:  reflect.TypeFor[falconv1alpha1.FalconAdmissionConfigSpec](),
			fields: []string{
				"admissionControlEnabled",
				"configMapWatcherEnabled",
				"containerPort",
				"deployWatcher",
				"disabledNamespaces",
				"failurePolicy",
				"falconImageAnalyzerNamespace",
				"imagePullPolicy",
				"imagePullSecrets",
				"nodeAffinity",
				"replicas",
				"resources",
				"resourcesClient",
				"resourcesClientNoWebhook",
				"resourcesWatcher",
				"serviceAccount",
				"servicePort",
				"snapshotsEnabled",
				"snapshotsInterval",
				"tls",
				"tolerations",
				"updateStrategy",
				"watcherEnabled",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertFieldsCovered(t, tt.name, tt.typ, tt.fields)
		})
	}
}
