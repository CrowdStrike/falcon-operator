package v1alpha1_test

// The tests in this file guard hack/migration/migrate_to_clusterguard.py, which migrates
// FalconDeployment, FalconNodeSensor, FalconAdmission and FalconImageAnalyzer manifests to
// FalconClusterGuard.
//
// TestMigrationScriptFieldCoverage fails when a field is added to or removed from any struct the
// script reads or writes. Review the script's handling of the field, then update the known-fields
// list below.
//
// TestMigrationScriptVerbatimCopies fails when a field the script copies as-is no longer has the
// same JSON shape on both sides, e.g. a nested field that exists only on the source and would be
// rejected by the FalconClusterGuard CRD.

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
)

const migrationScript = "hack/migration/migrate_to_clusterguard.py"

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
				"Update %s and the knownFields list in this test.",
			structName, uncovered, migrationScript,
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
		// Sources read by the migration script.
		{
			name: "FalconDeploymentSpec",
			typ:  reflect.TypeFor[falconv1alpha1.FalconDeploymentSpec](),
			fields: []string{
				"deployAdmissionController",
				"deployClusterGuard",
				"deployContainerSensor",
				"deployImageAnalyzer",
				"deployNodeSensor",
				"falconAdmission",
				"falconClusterGuard",
				"falconContainerSensor",
				"falconImageAnalyzer",
				"falconNodeSensor",
				"falconSecret",
				"falcon_api",
				"registry",
			},
		},
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
			name:   "FalconAdmissionRQSpec",
			typ:    reflect.TypeFor[falconv1alpha1.FalconAdmissionRQSpec](),
			fields: []string{"pods"},
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
		{
			name: "FalconImageAnalyzerSpec",
			typ:  reflect.TypeFor[falconv1alpha1.FalconImageAnalyzerSpec](),
			fields: []string{
				"falconSecret",
				"falcon_api",
				"image",
				"imageAnalyzerConfig",
				"installNamespace",
				"nodeAffinity",
				"registry",
				"tolerations",
				"version",
			},
		},
		{
			name: "FalconImageAnalyzerConfigSpec",
			typ:  reflect.TypeFor[falconv1alpha1.FalconImageAnalyzerConfigSpec](),
			fields: []string{
				"azureConfigPath",
				"clusterName",
				"debug",
				"exclusions",
				"iarAgentService",
				"imagePullPolicy",
				"imagePullSecrets",
				"kac",
				"logVerbosity",
				"mountPath",
				"priorityClass",
				"registryConfig",
				"resources",
				"serviceAccount",
				"sizeLimit",
				"updateStrategy",
			},
		},
		{
			name:   "RegistrySpec",
			typ:    reflect.TypeFor[falconv1alpha1.RegistrySpec](),
			fields: []string{"acr_name", "tls", "type"},
		},
		// Destinations written by the migration script. A new field here may have a source the
		// script should migrate.
		{
			name: "FalconClusterGuardSpec",
			typ:  reflect.TypeFor[falconv1alpha1.FalconClusterGuardSpec](),
			fields: []string{
				"controller",
				"falcon",
				"falconSecret",
				"falcon_api",
				"image",
				"imageAnalyzer",
				"imagePullPolicy",
				"imagePullSecrets",
				"installNamespace",
				"internal",
				"nodeSensor",
				"registry",
				"version",
			},
		},
		{
			name: "FalconClusterGuardNodeSpec",
			typ:  reflect.TypeFor[falconv1alpha1.FalconClusterGuardNodeSpec](),
			fields: []string{
				"advanced",
				"backend",
				"clusterName",
				"disableCleanup",
				"dnsConfig",
				"enabled",
				"gke",
				"guardian",
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
			name: "FalconClusterGuardController",
			typ:  reflect.TypeFor[falconv1alpha1.FalconClusterGuardController](),
			fields: []string{
				"admissionControlEnabled",
				"clusterName",
				"configMapWatcherEnabled",
				"containerPort",
				"disableClusterGuardController",
				"disabledNamespaces",
				"failurePolicy",
				"nodeAffinity",
				"replicas",
				"resourceQuotaPods",
				"resources",
				"resourcesClient",
				"resourcesClientNoWebhook",
				"resourcesWatcher",
				"serviceAccount",
				"servicePort",
				"snapshotsEnabled",
				"snapshotsInterval",
				"tls",
				"tlsVersionMinimum",
				"tolerations",
				"updateStrategy",
				"watcherEnabled",
				"watcherPort",
			},
		},
		{
			name: "FalconClusterGuardImageAnalyzerSpec",
			typ:  reflect.TypeFor[falconv1alpha1.FalconClusterGuardImageAnalyzerSpec](),
			fields: []string{
				"azureConfigPath",
				"clusterName",
				"debug",
				"enabled",
				"exclusions",
				"iarAgentService",
				"logVerbosity",
				"mountPath",
				"nodeAffinity",
				"priorityClass",
				"registryConfig",
				"resources",
				"serviceAccount",
				"sizeLimit",
				"tolerations",
				"updateStrategy",
			},
		},
		{
			name:   "FalconClusterGuardRegistrySpec",
			typ:    reflect.TypeFor[falconv1alpha1.FalconClusterGuardRegistrySpec](),
			fields: []string{"tls", "type"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertFieldsCovered(t, tt.name, tt.typ, tt.fields)
		})
	}
}

var apiPkgPath = reflect.TypeFor[falconv1alpha1.FalconClusterGuardSpec]().PkgPath()

// jsonField returns the type of the field serialized as name, looking through inlined embedded structs.
func jsonField(t reflect.Type, name string) (reflect.Type, bool) {
	for field := range t.Fields() {
		tagName, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		if field.Anonymous && tagName == "" {
			if ft, ok := jsonField(field.Type, name); ok {
				return ft, true
			}
			continue
		}
		if tagName == name {
			return field.Type, true
		}
	}
	return nil, false
}

// jsonShape records the JSON paths under prefix and the type at each leaf. Types from other packages
// are leaves, and this package's named scalar types reduce to their kind so that distinct string enums
// with the same values compare equal.
func jsonShape(t reflect.Type, prefix string, out map[string]string) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch {
	case t.Kind() == reflect.Slice || t.Kind() == reflect.Array:
		jsonShape(t.Elem(), prefix+"[]", out)
	case t.Kind() == reflect.Struct && t.PkgPath() == apiPkgPath:
		for field := range t.Fields() {
			tagName, _, _ := strings.Cut(field.Tag.Get("json"), ",")
			switch {
			case field.Anonymous && tagName == "":
				jsonShape(field.Type, prefix, out)
			case tagName != "" && tagName != "-":
				jsonShape(field.Type, prefix+"."+tagName, out)
			}
		}
	case t.PkgPath() == "" || (t.PkgPath() == apiPkgPath && t.Kind() != reflect.Map):
		out[prefix] = t.Kind().String()
	default:
		out[prefix] = t.String()
	}
}

func TestMigrationScriptVerbatimCopies(t *testing.T) {
	tests := []struct {
		name   string
		src    reflect.Type
		dst    reflect.Type
		fields []string
		// stripped lists source paths the script removes before copying.
		stripped []string
	}{
		{
			name:     "FalconNodeSensorSpec to FalconClusterGuardSpec",
			src:      reflect.TypeFor[falconv1alpha1.FalconNodeSensorSpec](),
			dst:      reflect.TypeFor[falconv1alpha1.FalconClusterGuardSpec](),
			fields:   []string{"installNamespace", "falcon"},
			stripped: []string{"falcon.cloud"},
		},
		// Credentials move to the FalconDeployment top level, which the controller passes to FalconClusterGuard.
		{
			name:   "FalconNodeSensorSpec to FalconDeploymentSpec",
			src:    reflect.TypeFor[falconv1alpha1.FalconNodeSensorSpec](),
			dst:    reflect.TypeFor[falconv1alpha1.FalconDeploymentSpec](),
			fields: []string{"falcon_api", "falconSecret"},
		},
		{
			name:   "FalconNodeSensorConfig to FalconClusterGuardSpec",
			src:    reflect.TypeFor[falconv1alpha1.FalconNodeSensorConfig](),
			dst:    reflect.TypeFor[falconv1alpha1.FalconClusterGuardSpec](),
			fields: []string{"imagePullPolicy", "imagePullSecrets"},
		},
		{
			name: "FalconNodeSensorConfig to FalconClusterGuardNodeSpec",
			src:  reflect.TypeFor[falconv1alpha1.FalconNodeSensorConfig](),
			dst:  reflect.TypeFor[falconv1alpha1.FalconClusterGuardNodeSpec](),
			fields: []string{
				"tolerations", "nodeAffinity", "updateStrategy", "terminationGracePeriod", "serviceAccount",
				"disableCleanup", "resources", "backend", "gke", "priorityClass", "advanced", "clusterName",
			},
		},
		{
			name:   "FalconAdmissionSpec to FalconClusterGuardSpec",
			src:    reflect.TypeFor[falconv1alpha1.FalconAdmissionSpec](),
			dst:    reflect.TypeFor[falconv1alpha1.FalconClusterGuardSpec](),
			fields: []string{"installNamespace", "falcon"},
		},
		{
			name:   "FalconAdmissionSpec to FalconDeploymentSpec",
			src:    reflect.TypeFor[falconv1alpha1.FalconAdmissionSpec](),
			dst:    reflect.TypeFor[falconv1alpha1.FalconDeploymentSpec](),
			fields: []string{"falcon_api", "falconSecret"},
		},
		{
			name:   "FalconAdmissionSpec to FalconClusterGuardController",
			src:    reflect.TypeFor[falconv1alpha1.FalconAdmissionSpec](),
			dst:    reflect.TypeFor[falconv1alpha1.FalconClusterGuardController](),
			fields: []string{"clusterName"},
		},
		{
			name:   "FalconAdmissionConfigSpec to FalconClusterGuardSpec",
			src:    reflect.TypeFor[falconv1alpha1.FalconAdmissionConfigSpec](),
			dst:    reflect.TypeFor[falconv1alpha1.FalconClusterGuardSpec](),
			fields: []string{"imagePullPolicy", "imagePullSecrets"},
		},
		{
			name: "FalconAdmissionConfigSpec to FalconClusterGuardController",
			src:  reflect.TypeFor[falconv1alpha1.FalconAdmissionConfigSpec](),
			dst:  reflect.TypeFor[falconv1alpha1.FalconClusterGuardController](),
			fields: []string{
				"serviceAccount", "servicePort", "containerPort", "tls", "failurePolicy", "disabledNamespaces",
				"watcherEnabled", "snapshotsEnabled", "snapshotsInterval", "admissionControlEnabled",
				"configMapWatcherEnabled", "replicas", "nodeAffinity", "tolerations", "updateStrategy",
				"resources", "resourcesClient", "resourcesClientNoWebhook", "resourcesWatcher",
			},
		},
		{
			name:   "FalconImageAnalyzerSpec to FalconClusterGuardSpec",
			src:    reflect.TypeFor[falconv1alpha1.FalconImageAnalyzerSpec](),
			dst:    reflect.TypeFor[falconv1alpha1.FalconClusterGuardSpec](),
			fields: []string{"installNamespace"},
		},
		{
			name:   "FalconImageAnalyzerSpec to FalconDeploymentSpec",
			src:    reflect.TypeFor[falconv1alpha1.FalconImageAnalyzerSpec](),
			dst:    reflect.TypeFor[falconv1alpha1.FalconDeploymentSpec](),
			fields: []string{"falcon_api", "falconSecret"},
		},
		{
			name:   "FalconImageAnalyzerSpec to FalconClusterGuardImageAnalyzerSpec",
			src:    reflect.TypeFor[falconv1alpha1.FalconImageAnalyzerSpec](),
			dst:    reflect.TypeFor[falconv1alpha1.FalconClusterGuardImageAnalyzerSpec](),
			fields: []string{"nodeAffinity", "tolerations"},
		},
		{
			name:   "FalconImageAnalyzerConfigSpec to FalconClusterGuardSpec",
			src:    reflect.TypeFor[falconv1alpha1.FalconImageAnalyzerConfigSpec](),
			dst:    reflect.TypeFor[falconv1alpha1.FalconClusterGuardSpec](),
			fields: []string{"imagePullPolicy", "imagePullSecrets"},
		},
		{
			name: "FalconImageAnalyzerConfigSpec to FalconClusterGuardImageAnalyzerSpec",
			src:  reflect.TypeFor[falconv1alpha1.FalconImageAnalyzerConfigSpec](),
			dst:  reflect.TypeFor[falconv1alpha1.FalconClusterGuardImageAnalyzerSpec](),
			fields: []string{
				"serviceAccount", "resources", "azureConfigPath", "priorityClass", "updateStrategy", "sizeLimit",
				"mountPath", "clusterName", "exclusions", "registryConfig", "debug", "logVerbosity", "iarAgentService",
			},
		},
		{
			name:   "RegistrySpec to FalconClusterGuardRegistrySpec",
			src:    reflect.TypeFor[falconv1alpha1.RegistrySpec](),
			dst:    reflect.TypeFor[falconv1alpha1.FalconClusterGuardRegistrySpec](),
			fields: []string{"tls"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src, dst := map[string]string{}, map[string]string{}
			for _, name := range tt.fields {
				srcField, ok := jsonField(tt.src, name)
				if !ok {
					t.Errorf("%s has no field %q", tt.src.Name(), name)
					continue
				}
				dstField, ok := jsonField(tt.dst, name)
				if !ok {
					t.Errorf("%s has no field %q", tt.dst.Name(), name)
					continue
				}
				jsonShape(srcField, name, src)
				jsonShape(dstField, name, dst)
			}

			stripped := make(map[string]bool, len(tt.stripped))
			for _, path := range tt.stripped {
				stripped[path] = true
				if _, ok := src[path]; !ok {
					t.Errorf("stripped path %q no longer exists on %s; update %s and this test",
						path, tt.src.Name(), migrationScript)
				}
			}

			paths := make([]string, 0, len(src))
			for path := range src {
				paths = append(paths, path)
			}
			sort.Strings(paths)
			for _, path := range paths {
				if stripped[path] {
					continue
				}
				dstType, ok := dst[path]
				switch {
				case !ok:
					t.Errorf("%s.%s has no counterpart on %s; %s copies it verbatim",
						tt.src.Name(), path, tt.dst.Name(), migrationScript)
				case dstType != src[path]:
					t.Errorf("%s.%s is %s but %s.%s is %s; %s copies it verbatim",
						tt.src.Name(), path, src[path], tt.dst.Name(), path, dstType, migrationScript)
				}
			}
		})
	}
}
