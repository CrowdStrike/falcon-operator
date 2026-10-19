package image_analyzer

import (
	"testing"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/internal/controller/components"
	pkgcommon "github.com/crowdstrike/falcon-operator/pkg/common"
)

func newIAForConfigMap(cid, namespace string, spec falconv1alpha1.FalconClusterGuardImageAnalyzerSpec, api *falconv1alpha1.FalconAPI) *ImageAnalyzer {
	return New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: namespace,
			Cid:              cid,
		},
		ImageAnalyzerSpec: spec,
		FalconAPI:         api,
	})
}

func TestBuildConfigMap_RequiredKeys(t *testing.T) {
	ia := newIAForConfigMap("abc123-xx", "falcon-clusterguard", falconv1alpha1.FalconClusterGuardImageAnalyzerSpec{}, nil)
	cm, err := ia.buildConfigMap()
	if err != nil {
		t.Fatalf("buildConfigMap returned error: %v", err)
	}
	if cm == nil {
		t.Fatal("expected non-nil ConfigMap")
	}

	required := []string{
		"AGENT_HELM_VERSION",
		"AGENT_DEBUG",
		"LOG_VERBOSITY",
		"SECRETS_AUTODISCOVER_ENABLED",
		"IAR_AGENT_SERVICE_PORT",
		"IS_KUBERNETES",
		"AGENT_CID",
		"AGENT_RUNMODE",
		"AGENT_MAX_CONSUMER_THREADS",
		"AGENT_TEMP_MOUNT_SIZE",
		"__CS_KAC_NAMESPACE",
	}
	for _, k := range required {
		if _, ok := cm.Data[k]; !ok {
			t.Errorf("expected ConfigMap key %q to be present", k)
		}
	}
}

func TestBuildConfigMap_CIDIsSet(t *testing.T) {
	ia := newIAForConfigMap("cid123-ab", "falcon-clusterguard", falconv1alpha1.FalconClusterGuardImageAnalyzerSpec{}, nil)
	cm, _ := ia.buildConfigMap()
	if cm.Data["AGENT_CID"] != "cid123-ab" {
		t.Errorf("expected AGENT_CID=cid123-ab, got %q", cm.Data["AGENT_CID"])
	}
}

func TestBuildConfigMap_KACNamespaceIsInstallNamespace(t *testing.T) {
	ia := newIAForConfigMap("", "my-namespace", falconv1alpha1.FalconClusterGuardImageAnalyzerSpec{}, nil)
	cm, _ := ia.buildConfigMap()
	if cm.Data["__CS_KAC_NAMESPACE"] != "my-namespace" {
		t.Errorf("expected __CS_KAC_NAMESPACE=my-namespace, got %q", cm.Data["__CS_KAC_NAMESPACE"])
	}
}

func TestBuildConfigMap_FalconAPICredentials(t *testing.T) {
	api := &falconv1alpha1.FalconAPI{
		ClientId:     "client-id",
		ClientSecret: "client-secret",
		CloudRegion:  "us-1",
	}
	ia := newIAForConfigMap("", "ns", falconv1alpha1.FalconClusterGuardImageAnalyzerSpec{}, api)
	cm, _ := ia.buildConfigMap()
	if cm.Data["AGENT_CLIENT_ID"] != "client-id" {
		t.Errorf("expected AGENT_CLIENT_ID=client-id, got %q", cm.Data["AGENT_CLIENT_ID"])
	}
	if cm.Data["AGENT_CLIENT_SECRET"] != "client-secret" {
		t.Errorf("expected AGENT_CLIENT_SECRET=client-secret, got %q", cm.Data["AGENT_CLIENT_SECRET"])
	}
	if cm.Data["AGENT_REGION"] == "" {
		t.Error("expected AGENT_REGION to be set")
	}
}

func TestBuildConfigMap_NoFalconAPICredentialsWhenNil(t *testing.T) {
	ia := newIAForConfigMap("", "ns", falconv1alpha1.FalconClusterGuardImageAnalyzerSpec{}, nil)
	cm, _ := ia.buildConfigMap()
	if _, ok := cm.Data["AGENT_CLIENT_ID"]; ok {
		t.Error("expected AGENT_CLIENT_ID to be absent when FalconAPI is nil")
	}
	if _, ok := cm.Data["AGENT_CLIENT_SECRET"]; ok {
		t.Error("expected AGENT_CLIENT_SECRET to be absent when FalconAPI is nil")
	}
}

func TestBuildConfigMap_ClusterName(t *testing.T) {
	spec := falconv1alpha1.FalconClusterGuardImageAnalyzerSpec{ClusterName: "my-cluster"}
	ia := newIAForConfigMap("", "ns", spec, nil)
	cm, _ := ia.buildConfigMap()
	if cm.Data["AGENT_CLUSTER_NAME"] != "my-cluster" {
		t.Errorf("expected AGENT_CLUSTER_NAME=my-cluster, got %q", cm.Data["AGENT_CLUSTER_NAME"])
	}
}

func TestBuildConfigMap_NoClusterNameKeyWhenEmpty(t *testing.T) {
	ia := newIAForConfigMap("", "ns", falconv1alpha1.FalconClusterGuardImageAnalyzerSpec{}, nil)
	cm, _ := ia.buildConfigMap()
	if _, ok := cm.Data["AGENT_CLUSTER_NAME"]; ok {
		t.Error("expected AGENT_CLUSTER_NAME to be absent when ClusterName is empty")
	}
}

func TestBuildConfigMap_NamespaceExclusions(t *testing.T) {
	spec := falconv1alpha1.FalconClusterGuardImageAnalyzerSpec{
		Exclusions: falconv1alpha1.Exclusions{Namespaces: []string{"kube-system", "default"}},
	}
	ia := newIAForConfigMap("", "ns", spec, nil)
	cm, _ := ia.buildConfigMap()
	if cm.Data["AGENT_NAMESPACE_EXCLUSIONS"] != "kube-system,default" {
		t.Errorf("expected AGENT_NAMESPACE_EXCLUSIONS=kube-system,default, got %q", cm.Data["AGENT_NAMESPACE_EXCLUSIONS"])
	}
}

func TestBuildConfigMap_VolumeSizeLimitDefault(t *testing.T) {
	ia := newIAForConfigMap("", "ns", falconv1alpha1.FalconClusterGuardImageAnalyzerSpec{}, nil)
	cm, _ := ia.buildConfigMap()
	if cm.Data["AGENT_TEMP_MOUNT_SIZE"] != "20Gi" {
		t.Errorf("expected AGENT_TEMP_MOUNT_SIZE=20Gi (default), got %q", cm.Data["AGENT_TEMP_MOUNT_SIZE"])
	}
}

func TestBuildConfigMap_VolumeSizeLimitCustom(t *testing.T) {
	spec := falconv1alpha1.FalconClusterGuardImageAnalyzerSpec{VolumeSizeLimit: "50Gi"}
	ia := newIAForConfigMap("", "ns", spec, nil)
	cm, _ := ia.buildConfigMap()
	if cm.Data["AGENT_TEMP_MOUNT_SIZE"] != "50Gi" {
		t.Errorf("expected AGENT_TEMP_MOUNT_SIZE=50Gi, got %q", cm.Data["AGENT_TEMP_MOUNT_SIZE"])
	}
}

func TestBuildConfigMap_Name(t *testing.T) {
	ia := newIAForConfigMap("", "ns", falconv1alpha1.FalconClusterGuardImageAnalyzerSpec{}, nil)
	cm, _ := ia.buildConfigMap()
	if cm.Name != pkgcommon.FCGImageAnalyzerConfigMapName {
		t.Errorf("expected ConfigMap name %q, got %q", pkgcommon.FCGImageAnalyzerConfigMapName, cm.Name)
	}
}

func TestBuildConfigMap_Namespace(t *testing.T) {
	ia := newIAForConfigMap("", "test-ns", falconv1alpha1.FalconClusterGuardImageAnalyzerSpec{}, nil)
	cm, _ := ia.buildConfigMap()
	if cm.Namespace != "test-ns" {
		t.Errorf("expected ConfigMap namespace test-ns, got %q", cm.Namespace)
	}
}
