package node_sensor

import (
	"testing"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/internal/controller/components"
	pkgcommon "github.com/crowdstrike/falcon-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
)

func TestClusterGuardSensorDaemonSetReturnsDS(t *testing.T) {
	expectedName := "falcon-sensor"
	cfg := Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
			Image:            "quay.io/crowdstrike/falcon-sensor:latest",
		},
	}

	n := New(nil, cfg)
	ds := n.daemonSet()

	if ds.Name != expectedName {
		t.Errorf("expected name %q, got %q", expectedName, ds.Name)
	}
	if ds.Namespace != cfg.InstallNamespace {
		t.Errorf("expected namespace %q, got %q", cfg.InstallNamespace, ds.Namespace)
	}
	if len(ds.Spec.Template.Spec.Containers) != 1 {
		t.Errorf("expected 1 container, got %d", len(ds.Spec.Template.Spec.Containers))
	}
	if ds.Spec.Template.Spec.Containers[0].Image != cfg.Image {
		t.Errorf("expected image %q, got %q", cfg.Image, ds.Spec.Template.Spec.Containers[0].Image)
	}
	// The container name and app label match the falcon-clusterguard Helm chart.
	if got := ds.Spec.Template.Spec.Containers[0].Name; got != "falcon-node-sensor" {
		t.Errorf("expected container falcon-node-sensor, got %q", got)
	}
	if got := ds.Spec.Template.Labels["app"]; got != "falcon-sensor" {
		t.Errorf("expected pod label app=falcon-sensor, got %q", got)
	}
	if got, want := ds.Spec.Template.Spec.ServiceAccountName, n.serviceAccount().Name; got != want {
		t.Errorf("DaemonSet uses ServiceAccount %q, but the node sensor creates %q", got, want)
	}
	// The TLS secret is created by the FalconClusterGuard controller (reconcileAPITLSSecrets), not a node sensor builder.
	foundTLS := false
	for _, v := range ds.Spec.Template.Spec.Volumes {
		if v.Secret != nil && v.Secret.SecretName == pkgcommon.ClusterGuardNodeSensorTLSSecretName {
			foundTLS = true
		}
	}
	if !foundTLS {
		t.Errorf("expected TLS secret %q in volumes, not found", pkgcommon.ClusterGuardNodeSensorTLSSecretName)
	}
}

func TestClusterGuardSensorDaemonSetDefaultTerminationGracePeriod(t *testing.T) {
	cfg := Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
			Image:            "quay.io/crowdstrike/falcon-sensor:latest",
		},
		// NodeSensor.TerminationGracePeriod is 0 (zero value) -> should default to 60
	}

	n := New(nil, cfg)
	ds := n.daemonSet()

	if ds.Spec.Template.Spec.TerminationGracePeriodSeconds == nil {
		t.Fatal("expected non-nil TerminationGracePeriodSeconds")
	}
	if *ds.Spec.Template.Spec.TerminationGracePeriodSeconds != 60 {
		t.Errorf("expected 60, got %d", *ds.Spec.Template.Spec.TerminationGracePeriodSeconds)
	}
}

func TestClusterGuardSensorCleanupDaemonSetReturnsDS(t *testing.T) {
	expectedName := "falcon-sensor-cleanup"
	cfg := Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
			Image:            "quay.io/crowdstrike/falcon-sensor:latest",
			ImagePullPolicy:  corev1.PullIfNotPresent,
			ImagePullSecrets: []corev1.LocalObjectReference{{Name: "mysecret"}},
		},
	}

	n := New(nil, cfg)
	ds := n.cleanupDaemonSet()

	if ds.Name != expectedName {
		t.Errorf("expected name %q, got %q", expectedName, ds.Name)
	}
	if ds.Namespace != cfg.InstallNamespace {
		t.Errorf("expected namespace %q, got %q", cfg.InstallNamespace, ds.Namespace)
	}
	if len(ds.Spec.Template.Spec.InitContainers) != 1 {
		t.Errorf("expected 1 init container, got %d", len(ds.Spec.Template.Spec.InitContainers))
	}
}

// dsResources tests

func TestDsResourcesNonBpfBackendReturnsEmpty(t *testing.T) {
	cfg := Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
			Image:            "quay.io/crowdstrike/falcon-sensor:latest",
		},
		NodeSensor: falconv1alpha1.FalconClusterGuardNodeSpec{
			Backend: "kernel",
		},
	}
	n := New(nil, cfg)
	ds := n.daemonSet()
	resources := ds.Spec.Template.Spec.Containers[0].Resources
	if len(resources.Limits) != 0 || len(resources.Requests) != 0 {
		t.Errorf("expected empty resources for non-bpf backend, got limits=%v requests=%v",
			resources.Limits, resources.Requests)
	}
}

func TestDsResourcesEmptyBackendReturnsEmpty(t *testing.T) {
	cfg := Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
			Image:            "quay.io/crowdstrike/falcon-sensor:latest",
		},
		NodeSensor: falconv1alpha1.FalconClusterGuardNodeSpec{},
	}
	n := New(nil, cfg)
	ds := n.daemonSet()
	resources := ds.Spec.Template.Spec.Containers[0].Resources
	if len(resources.Limits) != 0 || len(resources.Requests) != 0 {
		t.Errorf("expected empty resources when backend is empty, got limits=%v requests=%v",
			resources.Limits, resources.Requests)
	}
}

func TestDsResourcesBpfNoGKENoUserResourcesReturnsEmpty(t *testing.T) {
	cfg := Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
			Image:            "quay.io/crowdstrike/falcon-sensor:latest",
		},
		NodeSensor: falconv1alpha1.FalconClusterGuardNodeSpec{
			Backend: "bpf",
		},
	}
	n := New(nil, cfg)
	ds := n.daemonSet()
	resources := ds.Spec.Template.Spec.Containers[0].Resources
	if len(resources.Limits) != 0 || len(resources.Requests) != 0 {
		t.Errorf("expected empty resources for bpf without GKE or user resources, got limits=%v requests=%v",
			resources.Limits, resources.Requests)
	}
}

func TestDsResourcesBpfGKEEnabledSetsDefaults(t *testing.T) {
	gkeEnabled := true
	cfg := Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
			Image:            "quay.io/crowdstrike/falcon-sensor:latest",
		},
		NodeSensor: falconv1alpha1.FalconClusterGuardNodeSpec{
			Backend: "bpf",
			GKE:     falconv1alpha1.FalconClusterGuardAutoPilot{Enabled: &gkeEnabled},
		},
	}
	n := New(nil, cfg)
	ds := n.daemonSet()
	resources := ds.Spec.Template.Spec.Containers[0].Resources

	wantCPU := resource.MustParse("750m")
	wantMem := resource.MustParse("1.5Gi")
	wantEph := resource.MustParse("100Mi")

	if got := resources.Limits[corev1.ResourceCPU]; got.Cmp(wantCPU) != 0 {
		t.Errorf("GKE default CPU limit: want %s, got %s", wantCPU.String(), got.String())
	}
	if got := resources.Limits[corev1.ResourceMemory]; got.Cmp(wantMem) != 0 {
		t.Errorf("GKE default memory limit: want %s, got %s", wantMem.String(), got.String())
	}
	if got := resources.Limits[corev1.ResourceEphemeralStorage]; got.Cmp(wantEph) != 0 {
		t.Errorf("GKE default ephemeral-storage limit: want %s, got %s", wantEph.String(), got.String())
	}
	if got := resources.Requests[corev1.ResourceCPU]; got.Cmp(wantCPU) != 0 {
		t.Errorf("GKE default CPU request: want %s, got %s", wantCPU.String(), got.String())
	}
	if got := resources.Requests[corev1.ResourceMemory]; got.Cmp(wantMem) != 0 {
		t.Errorf("GKE default memory request: want %s, got %s", wantMem.String(), got.String())
	}
}

func TestDsResourcesBpfUserResourcesOverrideGKEDefaults(t *testing.T) {
	gkeEnabled := true
	cfg := Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
			Image:            "quay.io/crowdstrike/falcon-sensor:latest",
		},
		NodeSensor: falconv1alpha1.FalconClusterGuardNodeSpec{
			Backend: "bpf",
			GKE:     falconv1alpha1.FalconClusterGuardAutoPilot{Enabled: &gkeEnabled},
			SensorResources: falconv1alpha1.FalconClusterGuardResources{
				Limits:   falconv1alpha1.FalconClusterGuardResourceList{CPU: "1", Memory: "2Gi"},
				Requests: falconv1alpha1.FalconClusterGuardResourceList{CPU: "500m", Memory: "1Gi"},
			},
		},
	}
	n := New(nil, cfg)
	ds := n.daemonSet()
	resources := ds.Spec.Template.Spec.Containers[0].Resources

	wantLimitCPU := resource.MustParse("1")
	wantLimitMem := resource.MustParse("2Gi")
	wantReqCPU := resource.MustParse("500m")
	wantReqMem := resource.MustParse("1Gi")

	if got := resources.Limits[corev1.ResourceCPU]; got.Cmp(wantLimitCPU) != 0 {
		t.Errorf("user override CPU limit: want %s, got %s", wantLimitCPU.String(), got.String())
	}
	if got := resources.Limits[corev1.ResourceMemory]; got.Cmp(wantLimitMem) != 0 {
		t.Errorf("user override memory limit: want %s, got %s", wantLimitMem.String(), got.String())
	}
	if got := resources.Requests[corev1.ResourceCPU]; got.Cmp(wantReqCPU) != 0 {
		t.Errorf("user override CPU request: want %s, got %s", wantReqCPU.String(), got.String())
	}
	if got := resources.Requests[corev1.ResourceMemory]; got.Cmp(wantReqMem) != 0 {
		t.Errorf("user override memory request: want %s, got %s", wantReqMem.String(), got.String())
	}
}

func TestDsResourcesBpfUserResourcesWithoutGKE(t *testing.T) {
	cfg := Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
			Image:            "quay.io/crowdstrike/falcon-sensor:latest",
		},
		NodeSensor: falconv1alpha1.FalconClusterGuardNodeSpec{
			Backend: "bpf",
			SensorResources: falconv1alpha1.FalconClusterGuardResources{
				Limits:   falconv1alpha1.FalconClusterGuardResourceList{CPU: "500m", Memory: "512Mi", EphemeralStorage: "200Mi"},
				Requests: falconv1alpha1.FalconClusterGuardResourceList{CPU: "250m", Memory: "256Mi", EphemeralStorage: "100Mi"},
			},
		},
	}
	n := New(nil, cfg)
	ds := n.daemonSet()
	resources := ds.Spec.Template.Spec.Containers[0].Resources

	if _, ok := resources.Limits[corev1.ResourceCPU]; !ok {
		t.Error("expected CPU limit to be set")
	}
	if _, ok := resources.Limits[corev1.ResourceMemory]; !ok {
		t.Error("expected memory limit to be set")
	}
	if _, ok := resources.Limits[corev1.ResourceEphemeralStorage]; !ok {
		t.Error("expected ephemeral-storage limit to be set")
	}
	if _, ok := resources.Requests[corev1.ResourceCPU]; !ok {
		t.Error("expected CPU request to be set")
	}
	if _, ok := resources.Requests[corev1.ResourceMemory]; !ok {
		t.Error("expected memory request to be set")
	}
	if _, ok := resources.Requests[corev1.ResourceEphemeralStorage]; !ok {
		t.Error("expected ephemeral-storage request to be set")
	}
}

func TestDsResourcesBpfPartialUserOverride(t *testing.T) {
	gkeEnabled := true
	cfg := Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
			Image:            "quay.io/crowdstrike/falcon-sensor:latest",
		},
		NodeSensor: falconv1alpha1.FalconClusterGuardNodeSpec{
			Backend: "bpf",
			GKE:     falconv1alpha1.FalconClusterGuardAutoPilot{Enabled: &gkeEnabled},
			SensorResources: falconv1alpha1.FalconClusterGuardResources{
				// Only override CPU limit; memory should remain the GKE default
				Limits: falconv1alpha1.FalconClusterGuardResourceList{CPU: "2"},
			},
		},
	}
	n := New(nil, cfg)
	ds := n.daemonSet()
	resources := ds.Spec.Template.Spec.Containers[0].Resources

	wantCPU := resource.MustParse("2")
	wantMem := resource.MustParse("1.5Gi") // GKE default unchanged

	if got := resources.Limits[corev1.ResourceCPU]; got.Cmp(wantCPU) != 0 {
		t.Errorf("partial override CPU limit: want %s, got %s", wantCPU.String(), got.String())
	}
	if got := resources.Limits[corev1.ResourceMemory]; got.Cmp(wantMem) != 0 {
		t.Errorf("non-overridden memory limit should be GKE default: want %s, got %s", wantMem.String(), got.String())
	}
}

// configMap builder tests

func TestClusterGuardSensorDaemonSetVolumesAndMounts(t *testing.T) {
	n := New(nil, Config{BaseConfig: components.BaseConfig{
		InstallNamespace: "falcon-clusterguard",
		Image:            "quay.io/crowdstrike/falcon-sensor:latest",
	}})
	ds := n.daemonSet()

	wantVolumes := []string{"cs-config", "falconstore", "falcon-sensor-tls-certs", "falcon-api-ca"}
	volumeNames := map[string]bool{}
	for _, v := range ds.Spec.Template.Spec.Volumes {
		volumeNames[v.Name] = true
	}
	for _, name := range wantVolumes {
		if !volumeNames[name] {
			t.Errorf("missing volume %q", name)
		}
	}

	wantMounts := []string{"cs-config", "falconstore"}
	for _, want := range wantMounts {
		foundInit, foundMain := false, false
		for _, vm := range ds.Spec.Template.Spec.InitContainers[0].VolumeMounts {
			if vm.Name == want {
				foundInit = true
			}
		}
		for _, vm := range ds.Spec.Template.Spec.Containers[0].VolumeMounts {
			if vm.Name == want {
				foundMain = true
			}
		}
		if !foundInit {
			t.Errorf("init container missing volumeMount %q", want)
		}
		if !foundMain {
			t.Errorf("main container missing volumeMount %q", want)
		}
	}
}

func TestNodeSensorConfigMapName(t *testing.T) {
	cfg := Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
			Image:            "quay.io/crowdstrike/falcon-sensor:latest",
		},
		NodeSensor: falconv1alpha1.FalconClusterGuardNodeSpec{},
	}
	n := New(nil, cfg)
	ds := n.daemonSet()

	expectedCM := "falcon-sensor-config"
	found := false
	for _, ef := range ds.Spec.Template.Spec.Containers[0].EnvFrom {
		if ef.ConfigMapRef != nil && ef.ConfigMapRef.Name == expectedCM {
			found = true
		}
	}
	if !found {
		t.Errorf("expected container EnvFrom to reference ConfigMap %q", expectedCM)
	}
}

func TestNodeSensorConfigMapNameGKEUsesFixedName(t *testing.T) {
	gkeEnabled := true
	cfg := Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
			Image:            "quay.io/crowdstrike/falcon-sensor:latest",
		},
		NodeSensor: falconv1alpha1.FalconClusterGuardNodeSpec{
			GKE: falconv1alpha1.FalconClusterGuardAutoPilot{Enabled: &gkeEnabled},
		},
	}
	n := New(nil, cfg)
	ds := n.daemonSet()

	found := false
	for _, ef := range ds.Spec.Template.Spec.Containers[0].EnvFrom {
		if ef.ConfigMapRef != nil && ef.ConfigMapRef.Name == pkgcommon.GKEAutoPilotConfigMapName {
			found = true
		}
	}
	if !found {
		t.Errorf("expected GKE DaemonSet to reference fixed ConfigMap %q", pkgcommon.GKEAutoPilotConfigMapName)
	}
}

func TestNodeSensorConfigMapAPIServiceNameIncludesNamespace(t *testing.T) {
	namespace := "my-namespace"
	cfg := Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: namespace,
			Image:            "quay.io/crowdstrike/falcon-sensor:latest",
		},
	}
	n := New(nil, cfg)
	ds := n.daemonSet()

	expectedAPIServiceName := pkgcommon.ClusterGuardAPIServiceName + "." + namespace + ".svc"

	foundInit := false
	for _, env := range ds.Spec.Template.Spec.InitContainers[0].Env {
		if env.Name == "API_SERVICE_NAME" && env.Value == expectedAPIServiceName {
			foundInit = true
		}
	}
	if !foundInit {
		t.Errorf("init container: expected API_SERVICE_NAME=%q", expectedAPIServiceName)
	}

	foundMain := false
	for _, env := range ds.Spec.Template.Spec.Containers[0].Env {
		if env.Name == "API_SERVICE_NAME" && env.Value == expectedAPIServiceName {
			foundMain = true
		}
	}
	if !foundMain {
		t.Errorf("main container: expected API_SERVICE_NAME=%q", expectedAPIServiceName)
	}
}

func TestNodeSensorDaemonSetMainContainerHasHostIP(t *testing.T) {
	cfg := Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
			Image:            "quay.io/crowdstrike/falcon-sensor:latest",
		},
	}
	n := New(nil, cfg)
	ds := n.daemonSet()

	found := false
	for _, env := range ds.Spec.Template.Spec.Containers[0].Env {
		if env.Name == "HOST_IP" && env.ValueFrom != nil &&
			env.ValueFrom.FieldRef != nil &&
			env.ValueFrom.FieldRef.FieldPath == "status.hostIP" {
			found = true
		}
	}
	if !found {
		t.Error("expected main container to have HOST_IP env var from status.hostIP")
	}
}

// ConfigMap builder tests

func TestNodeSensorConfigMapHasRequiredKeys(t *testing.T) {
	cfg := Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
			Image:            "quay.io/crowdstrike/falcon-sensor:latest",
		},
	}
	n := New(nil, cfg)
	cm := n.configMap()

	requiredKeys := []string{
		"FALCONCTL_OPT_BACKEND",
		"FALCON_MODE",
		"__CS_ENABLE_K8S_METADATA_SERVICE",
	}
	for _, k := range requiredKeys {
		if _, ok := cm.Data[k]; !ok {
			t.Errorf("expected ConfigMap key %q to be present", k)
		}
	}
}

func TestNodeSensorConfigMapStaticValues(t *testing.T) {
	cfg := Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
			Image:            "quay.io/crowdstrike/falcon-sensor:latest",
		},
	}
	n := New(nil, cfg)
	cm := n.configMap()

	if cm.Data["FALCONCTL_OPT_BACKEND"] != "bpf" {
		t.Errorf("expected FALCONCTL_OPT_BACKEND=bpf, got %q", cm.Data["FALCONCTL_OPT_BACKEND"])
	}
	if cm.Data["FALCON_MODE"] != "daemonset" {
		t.Errorf("expected FALCON_MODE=daemonset, got %q", cm.Data["FALCON_MODE"])
	}
	if cm.Data["__CS_ENABLE_K8S_METADATA_SERVICE"] != "true" {
		t.Errorf("expected __CS_ENABLE_K8S_METADATA_SERVICE=true, got %q", cm.Data["__CS_ENABLE_K8S_METADATA_SERVICE"])
	}
}

func TestNodeSensorConfigMapCidOverride(t *testing.T) {
	cfg := Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
			Image:            "quay.io/crowdstrike/falcon-sensor:latest",
			Cid:              "deadbeef00000000000000000000000000000000-ab",
		},
	}
	n := New(nil, cfg)
	cm := n.configMap()

	if cm.Data["FALCONCTL_OPT_CID"] != "deadbeef00000000000000000000000000000000-ab" {
		t.Errorf("expected FALCONCTL_OPT_CID to be set from Cid, got %q", cm.Data["FALCONCTL_OPT_CID"])
	}
}

// Cluster-wide and shared resources keep fixed names.
func TestNodeSensorFixedNames(t *testing.T) {
	cfg := Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "test-ns",
		},
	}
	n := New(nil, cfg)

	tests := []struct {
		name          string
		gotName       string
		gotNamespace  string
		wantName      string
		wantNamespace string
	}{
		{"ConfigMap", n.configMap().Name, n.configMap().Namespace, pkgcommon.ClusterGuardNodeSensorConfigMapName, "test-ns"},
		{"ServiceAccount", n.serviceAccount().Name, n.serviceAccount().Namespace, pkgcommon.ClusterGuardNodeSensorServiceAccountName, "test-ns"},
		{"CleanupServiceAccount", n.cleanupServiceAccount().Name, n.cleanupServiceAccount().Namespace, pkgcommon.ClusterGuardNodeSensorCleanupServiceAccountName, "test-ns"},
		{"ClusterRoleBinding", n.clusterRoleBinding().Name, n.clusterRoleBinding().Namespace, pkgcommon.ClusterGuardNodeSensorClusterRoleBindingName, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.gotName != tt.wantName {
				t.Errorf("expected name %q, got %q", tt.wantName, tt.gotName)
			}
			if tt.gotNamespace != tt.wantNamespace {
				t.Errorf("expected namespace %q, got %q", tt.wantNamespace, tt.gotNamespace)
			}
		})
	}
}

// ServiceAccount builder tests

// assets.ServiceAccount is covered in the assets package; this only checks which
// config fields the node sensor feeds into it.
func TestNodeSensorServiceAccountWiring(t *testing.T) {
	cfg := Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
			ImagePullSecrets: []corev1.LocalObjectReference{{Name: "my-pull-secret"}},
		},
		NodeSensor: falconv1alpha1.FalconClusterGuardNodeSpec{
			ServiceAccount: falconv1alpha1.FalconClusterGuardNodeServiceAccount{
				Annotations: map[string]string{"iam.amazonaws.com/role": "arn:aws:iam::123:role/my-role"},
			},
		},
	}
	n := New(nil, cfg)
	sa := n.serviceAccount()

	if len(sa.ImagePullSecrets) != 1 || sa.ImagePullSecrets[0].Name != "my-pull-secret" {
		t.Errorf("expected ImagePullSecrets [my-pull-secret], got %v", sa.ImagePullSecrets)
	}
	if sa.Annotations["iam.amazonaws.com/role"] != "arn:aws:iam::123:role/my-role" {
		t.Errorf("expected NodeSensor.ServiceAccount annotations to be propagated, got %v", sa.Annotations)
	}
}

// ClusterRoleBinding builder tests

// assets.ClusterRoleBinding is covered in the assets package; this only checks the
// role, ServiceAccount and namespace the node sensor passes in.
func TestNodeSensorClusterRoleBindingWiring(t *testing.T) {
	cfg := Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "test-ns",
		},
	}
	n := New(nil, cfg)
	crb := n.clusterRoleBinding()

	if crb.RoleRef.Name != pkgcommon.ClusterGuardNodeSensorClusterRoleName {
		t.Errorf("expected RoleRef.Name=%q, got %q", pkgcommon.ClusterGuardNodeSensorClusterRoleName, crb.RoleRef.Name)
	}
	if len(crb.Subjects) != 1 {
		t.Fatalf("expected 1 subject, got %d", len(crb.Subjects))
	}
	subj := crb.Subjects[0]
	if subj.Name != pkgcommon.ClusterGuardNodeSensorServiceAccountName {
		t.Errorf("expected subject Name=%q, got %q", pkgcommon.ClusterGuardNodeSensorServiceAccountName, subj.Name)
	}
	if subj.Namespace != "test-ns" {
		t.Errorf("expected subject Namespace=%q, got %q", "test-ns", subj.Namespace)
	}
}

// CleanupServiceAccount builder tests

func TestNodeSensorCleanupServiceAccountHasNoImagePullSecrets(t *testing.T) {
	cfg := Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
			ImagePullSecrets: []corev1.LocalObjectReference{{Name: "my-secret"}},
		},
	}
	n := New(nil, cfg)
	sa := n.cleanupServiceAccount()

	// Cleanup SA intentionally does not inherit ImagePullSecrets
	if len(sa.ImagePullSecrets) != 0 {
		t.Errorf("expected cleanup SA to have no ImagePullSecrets, got %d", len(sa.ImagePullSecrets))
	}
}

// sensorCapabilities tests

func TestSensorCapabilitiesNilWhenGKEDisabled(t *testing.T) {
	cfg := Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
		},
	}
	n := New(nil, cfg)

	if caps := n.sensorCapabilities(false); caps != nil {
		t.Errorf("expected nil capabilities when GKE not enabled, got %+v", caps)
	}
	if caps := n.sensorCapabilities(true); caps != nil {
		t.Errorf("expected nil init capabilities when GKE not enabled, got %+v", caps)
	}
}

func TestSensorCapabilitiesNilWhenGKEEnabledFalse(t *testing.T) {
	gkeEnabled := false
	cfg := Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
		},
		NodeSensor: falconv1alpha1.FalconClusterGuardNodeSpec{GKE: falconv1alpha1.FalconClusterGuardAutoPilot{Enabled: &gkeEnabled}},
	}
	n := New(nil, cfg)

	if caps := n.sensorCapabilities(false); caps != nil {
		t.Errorf("expected nil capabilities when GKE.Enabled=false, got %+v", caps)
	}
}

func TestSensorCapabilitiesGKEMainContainerHasExpectedCaps(t *testing.T) {
	gkeEnabled := true
	cfg := Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
		},
		NodeSensor: falconv1alpha1.FalconClusterGuardNodeSpec{GKE: falconv1alpha1.FalconClusterGuardAutoPilot{Enabled: &gkeEnabled}},
	}
	n := New(nil, cfg)
	caps := n.sensorCapabilities(false)

	if caps == nil {
		t.Fatal("expected non-nil capabilities for GKE main container")
	}

	wantCaps := map[corev1.Capability]bool{
		"SYS_ADMIN":       true,
		"SETGID":          true,
		"SETUID":          true,
		"SYS_PTRACE":      true,
		"SYS_CHROOT":      true,
		"DAC_OVERRIDE":    true,
		"SETPCAP":         true,
		"DAC_READ_SEARCH": true,
		"BPF":             true,
		"PERFMON":         true,
		"SYS_RESOURCE":    true,
		"NET_RAW":         true,
		"CHOWN":           true,
		"NET_ADMIN":       true,
	}
	for _, got := range caps.Add {
		if !wantCaps[got] {
			t.Errorf("unexpected capability %q in GKE main container", got)
		}
		delete(wantCaps, got)
	}
	for missing := range wantCaps {
		t.Errorf("expected capability %q missing from GKE main container", missing)
	}
	if len(caps.Drop) != 0 {
		t.Errorf("expected no dropped capabilities, got %v", caps.Drop)
	}
}

func TestSensorCapabilitiesGKEInitContainerHasExpectedCaps(t *testing.T) {
	gkeEnabled := true
	cfg := Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
		},
		NodeSensor: falconv1alpha1.FalconClusterGuardNodeSpec{GKE: falconv1alpha1.FalconClusterGuardAutoPilot{Enabled: &gkeEnabled}},
	}
	n := New(nil, cfg)
	caps := n.sensorCapabilities(true)

	if caps == nil {
		t.Fatal("expected non-nil capabilities for GKE init container")
	}

	wantCaps := map[corev1.Capability]bool{
		"SYS_ADMIN":       true,
		"SYS_PTRACE":      true,
		"SYS_CHROOT":      true,
		"DAC_READ_SEARCH": true,
	}
	for _, got := range caps.Add {
		if !wantCaps[got] {
			t.Errorf("unexpected capability %q in GKE init container", got)
		}
		delete(wantCaps, got)
	}
	for missing := range wantCaps {
		t.Errorf("expected capability %q missing from GKE init container", missing)
	}
}

func TestSensorCapabilitiesGKEInitContainerFewerThanMainContainer(t *testing.T) {
	gkeEnabled := true
	cfg := Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
		},
		NodeSensor: falconv1alpha1.FalconClusterGuardNodeSpec{GKE: falconv1alpha1.FalconClusterGuardAutoPilot{Enabled: &gkeEnabled}},
	}
	n := New(nil, cfg)

	initCaps := n.sensorCapabilities(true)
	mainCaps := n.sensorCapabilities(false)

	if len(initCaps.Add) >= len(mainCaps.Add) {
		t.Errorf("expected init container to have fewer capabilities than main container: init=%d main=%d",
			len(initCaps.Add), len(mainCaps.Add))
	}
}

func TestSensorCapabilitiesAppearsInDaemonSetContainers(t *testing.T) {
	gkeEnabled := true
	cfg := Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
			Image:            "quay.io/crowdstrike/falcon-sensor:latest",
		},
		NodeSensor: falconv1alpha1.FalconClusterGuardNodeSpec{GKE: falconv1alpha1.FalconClusterGuardAutoPilot{Enabled: &gkeEnabled}},
	}
	n := New(nil, cfg)
	ds := n.daemonSet()

	mainCtx := ds.Spec.Template.Spec.Containers[0]
	if mainCtx.SecurityContext == nil || mainCtx.SecurityContext.Capabilities == nil {
		t.Fatal("expected main container to have capabilities set")
	}
	initCtx := ds.Spec.Template.Spec.InitContainers[0]
	if initCtx.SecurityContext == nil || initCtx.SecurityContext.Capabilities == nil {
		t.Fatal("expected init container to have capabilities set")
	}
}

// DSAutoPilotDeployAllowlistLabel tests

func TestDSAutoPilotDeployAllowlistLabelNilWhenGKEDisabled(t *testing.T) {
	cfg := Config{BaseConfig: components.BaseConfig{InstallNamespace: "falcon-clusterguard"}}
	n := New(nil, cfg)

	if labels := n.dsAutoPilotDeployAllowlistLabel(); labels != nil {
		t.Errorf("expected nil when GKE not enabled, got %v", labels)
	}
}

func TestDSAutoPilotDeployAllowlistLabelNilWhenVersionAbsent(t *testing.T) {
	gkeEnabled := true
	cfg := Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
		},
		NodeSensor: falconv1alpha1.FalconClusterGuardNodeSpec{GKE: falconv1alpha1.FalconClusterGuardAutoPilot{Enabled: &gkeEnabled}},
	}
	n := New(nil, cfg)

	// GKE enabled but no DeployAllowListVersion — should return nil
	if labels := n.dsAutoPilotDeployAllowlistLabel(); labels != nil {
		t.Errorf("expected nil when DeployAllowListVersion is absent, got %v", labels)
	}
}

func TestDSAutoPilotDeployAllowlistLabelValue(t *testing.T) {
	gkeEnabled := true
	version := "v1.2.3"
	cfg := Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
		},
		NodeSensor: falconv1alpha1.FalconClusterGuardNodeSpec{
			GKE: falconv1alpha1.FalconClusterGuardAutoPilot{Enabled: &gkeEnabled, DeployAllowListVersion: &version},
		},
	}
	n := New(nil, cfg)
	labels := n.dsAutoPilotDeployAllowlistLabel()

	if labels == nil {
		t.Fatal("expected non-nil label map")
	}
	val, ok := labels[pkgcommon.GKEAutoPilotAllowListLabelKey]
	if !ok {
		t.Fatalf("expected key %q in label map", pkgcommon.GKEAutoPilotAllowListLabelKey)
	}
	expected := pkgcommon.GKEAutoPilotDeployDSAllowlistPrefix + "-" + version
	if val != expected {
		t.Errorf("expected label value %q, got %q", expected, val)
	}
}

// DSAutoPilotCleanupAllowlistLabel tests

func TestDSAutoPilotCleanupAllowlistLabelNilWhenGKEDisabled(t *testing.T) {
	cfg := Config{BaseConfig: components.BaseConfig{InstallNamespace: "falcon-clusterguard"}}
	n := New(nil, cfg)

	if labels := n.dsAutoPilotCleanupAllowlistLabel(); labels != nil {
		t.Errorf("expected nil when GKE not enabled, got %v", labels)
	}
}

func TestDSAutoPilotCleanupAllowlistLabelNilWhenVersionAbsent(t *testing.T) {
	gkeEnabled := true
	cfg := Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
		},
		NodeSensor: falconv1alpha1.FalconClusterGuardNodeSpec{GKE: falconv1alpha1.FalconClusterGuardAutoPilot{Enabled: &gkeEnabled}},
	}
	n := New(nil, cfg)

	if labels := n.dsAutoPilotCleanupAllowlistLabel(); labels != nil {
		t.Errorf("expected nil when CleanupAllowListVersion is absent, got %v", labels)
	}
}

func TestDSAutoPilotCleanupAllowlistLabelValue(t *testing.T) {
	gkeEnabled := true
	version := "v2.0.0"
	cfg := Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
		},
		NodeSensor: falconv1alpha1.FalconClusterGuardNodeSpec{
			GKE: falconv1alpha1.FalconClusterGuardAutoPilot{Enabled: &gkeEnabled, CleanupAllowListVersion: &version},
		},
	}
	n := New(nil, cfg)
	labels := n.dsAutoPilotCleanupAllowlistLabel()

	if labels == nil {
		t.Fatal("expected non-nil label map")
	}
	val, ok := labels[pkgcommon.GKEAutoPilotAllowListLabelKey]
	if !ok {
		t.Fatalf("expected key %q in label map", pkgcommon.GKEAutoPilotAllowListLabelKey)
	}
	expected := pkgcommon.GKEAutoPilotCleanupAllowlistPrefix + "-" + version
	if val != expected {
		t.Errorf("expected label value %q, got %q", expected, val)
	}
}

// DSManageAutoPilotLabels tests

func TestDSManageAutoPilotLabelsIncludesAppLabel(t *testing.T) {
	cfg := Config{BaseConfig: components.BaseConfig{InstallNamespace: "falcon-clusterguard"}}
	n := New(nil, cfg)

	labels := n.dsManageAutoPilotLabels("daemonset", "my-ds", pkgcommon.ClusterGuardNodeSensorComponentName, func() map[string]string { return nil })

	if labels["app"] != "my-ds" {
		t.Errorf("expected app=my-ds, got %q", labels["app"])
	}
}

func TestDSManageAutoPilotLabelsNilLabelFuncProducesNoCrash(t *testing.T) {
	cfg := Config{BaseConfig: components.BaseConfig{InstallNamespace: "falcon-clusterguard"}}
	n := New(nil, cfg)

	labels := n.dsManageAutoPilotLabels("daemonset", "my-ds", pkgcommon.ClusterGuardNodeSensorComponentName, func() map[string]string { return nil })

	if labels == nil {
		t.Error("expected non-nil label map even when labelFunc returns nil")
	}
	if _, ok := labels[pkgcommon.GKEAutoPilotAllowListLabelKey]; ok {
		t.Error("expected no GKE autopilot label when labelFunc returns nil")
	}
}

func TestDSManageAutoPilotLabelsMergesAutoPilotLabel(t *testing.T) {
	cfg := Config{BaseConfig: components.BaseConfig{InstallNamespace: "falcon-clusterguard"}}
	n := New(nil, cfg)

	autoPilotLabels := map[string]string{pkgcommon.GKEAutoPilotAllowListLabelKey: "some-allowlist-v1.0.0"}
	labels := n.dsManageAutoPilotLabels("daemonset", "my-ds", pkgcommon.ClusterGuardNodeSensorComponentName, func() map[string]string { return autoPilotLabels })

	if labels[pkgcommon.GKEAutoPilotAllowListLabelKey] != "some-allowlist-v1.0.0" {
		t.Errorf("expected GKE autopilot label to be merged, got %q",
			labels[pkgcommon.GKEAutoPilotAllowListLabelKey])
	}
}

func TestDSManageAutoPilotLabelsAppearsInDaemonSetPodTemplate(t *testing.T) {
	gkeEnabled := true
	version := "v1.2.3"
	cfg := Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
			Image:            "quay.io/crowdstrike/falcon-sensor:latest",
		},
		NodeSensor: falconv1alpha1.FalconClusterGuardNodeSpec{
			GKE: falconv1alpha1.FalconClusterGuardAutoPilot{Enabled: &gkeEnabled, DeployAllowListVersion: &version},
		},
	}
	n := New(nil, cfg)
	ds := n.daemonSet()

	podLabels := ds.Spec.Template.ObjectMeta.Labels
	expectedVal := pkgcommon.GKEAutoPilotDeployDSAllowlistPrefix + "-" + version
	if podLabels[pkgcommon.GKEAutoPilotAllowListLabelKey] != expectedVal {
		t.Errorf("expected pod template label %q=%q, got %q",
			pkgcommon.GKEAutoPilotAllowListLabelKey, expectedVal,
			podLabels[pkgcommon.GKEAutoPilotAllowListLabelKey])
	}
}

func TestDSManageAutoPilotLabelsCleanupAppearsInCleanupDaemonSet(t *testing.T) {
	gkeEnabled := true
	version := "v2.0.0"
	cfg := Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
			Image:            "quay.io/crowdstrike/falcon-sensor:latest",
		},
		NodeSensor: falconv1alpha1.FalconClusterGuardNodeSpec{
			GKE: falconv1alpha1.FalconClusterGuardAutoPilot{Enabled: &gkeEnabled, CleanupAllowListVersion: &version},
		},
	}
	n := New(nil, cfg)
	ds := n.cleanupDaemonSet()

	podLabels := ds.Spec.Template.ObjectMeta.Labels
	expectedVal := pkgcommon.GKEAutoPilotCleanupAllowlistPrefix + "-" + version
	if podLabels[pkgcommon.GKEAutoPilotAllowListLabelKey] != expectedVal {
		t.Errorf("expected cleanup pod template label %q=%q, got %q",
			pkgcommon.GKEAutoPilotAllowListLabelKey, expectedVal,
			podLabels[pkgcommon.GKEAutoPilotAllowListLabelKey])
	}
}

// Guardian proxy builder tests

func TestProxyServiceBuilderName(t *testing.T) {
	n := New(nil, Config{BaseConfig: components.BaseConfig{InstallNamespace: "falcon-clusterguard"}})
	svc := n.proxyService()

	if svc.Name != pkgcommon.ClusterGuardProxyServiceName {
		t.Errorf("expected name %q, got %q", pkgcommon.ClusterGuardProxyServiceName, svc.Name)
	}
	if svc.Namespace != "falcon-clusterguard" {
		t.Errorf("expected namespace %q, got %q", "falcon-clusterguard", svc.Namespace)
	}
}

func TestProxyServiceBuilderInternalTrafficPolicyLocal(t *testing.T) {
	n := New(nil, Config{BaseConfig: components.BaseConfig{InstallNamespace: "falcon-clusterguard"}})
	svc := n.proxyService()

	if svc.Spec.InternalTrafficPolicy == nil {
		t.Fatal("expected non-nil InternalTrafficPolicy")
	}
	if *svc.Spec.InternalTrafficPolicy != corev1.ServiceInternalTrafficPolicyLocal {
		t.Errorf("expected InternalTrafficPolicy=Local, got %q", *svc.Spec.InternalTrafficPolicy)
	}
}

func TestProxyServiceSelectorMatchesNodeSensorPods(t *testing.T) {
	n := New(nil, Config{BaseConfig: components.BaseConfig{InstallNamespace: "falcon-clusterguard"}})
	selector := labels.SelectorFromSet(n.proxyService().Spec.Selector)

	if podLabels := n.daemonSet().Spec.Template.Labels; !selector.Matches(labels.Set(podLabels)) {
		t.Errorf("proxy Service selector %v does not match node sensor pod labels %v", selector, podLabels)
	}
	if podLabels := n.cleanupDaemonSet().Spec.Template.Labels; selector.Matches(labels.Set(podLabels)) {
		t.Errorf("proxy Service selector %v must not match cleanup pod labels %v", selector, podLabels)
	}
}

func TestDaemonSetSelectorsMatchOnlyTheirOwnPods(t *testing.T) {
	n := New(nil, Config{BaseConfig: components.BaseConfig{InstallNamespace: "falcon-clusterguard"}})
	sensor, cleanup := n.daemonSet(), n.cleanupDaemonSet()
	sensorSel := labels.SelectorFromSet(sensor.Spec.Selector.MatchLabels)
	cleanupSel := labels.SelectorFromSet(cleanup.Spec.Selector.MatchLabels)
	sensorPods, cleanupPods := labels.Set(sensor.Spec.Template.Labels), labels.Set(cleanup.Spec.Template.Labels)

	if !sensorSel.Matches(sensorPods) || sensorSel.Matches(cleanupPods) {
		t.Errorf("sensor DaemonSet selector %v must match only sensor pods; sensor=%v cleanup=%v", sensorSel, sensorPods, cleanupPods)
	}
	if !cleanupSel.Matches(cleanupPods) || cleanupSel.Matches(sensorPods) {
		t.Errorf("cleanup DaemonSet selector %v must match only cleanup pods; sensor=%v cleanup=%v", cleanupSel, sensorPods, cleanupPods)
	}
}

func TestProxyServiceBuilderPortMapsTo80(t *testing.T) {
	n := New(nil, Config{BaseConfig: components.BaseConfig{InstallNamespace: "falcon-clusterguard"}})
	svc := n.proxyService()

	if len(svc.Spec.Ports) != 1 {
		t.Fatalf("expected 1 port, got %d", len(svc.Spec.Ports))
	}
	p := svc.Spec.Ports[0]
	if p.Name != pkgcommon.ClusterGuardProxyPortName {
		t.Errorf("expected port name %q, got %q", pkgcommon.ClusterGuardProxyPortName, p.Name)
	}
	if p.Port != pkgcommon.ClusterGuardProxyServicePort {
		t.Errorf("expected port %d, got %d", pkgcommon.ClusterGuardProxyServicePort, p.Port)
	}
	if p.TargetPort.String() != pkgcommon.ClusterGuardProxyPortName {
		t.Errorf("expected targetPort %q, got %q", pkgcommon.ClusterGuardProxyPortName, p.TargetPort.String())
	}
}

func TestDaemonSetProxyDisabledByDefaultNoPortOrVolume(t *testing.T) {
	n := New(nil, Config{BaseConfig: components.BaseConfig{
		InstallNamespace: "falcon-clusterguard",
		Image:            "quay.io/crowdstrike/falcon-sensor:latest",
	}})
	ds := n.daemonSet()

	for _, p := range ds.Spec.Template.Spec.Containers[0].Ports {
		if p.Name == pkgcommon.ClusterGuardProxyPortName {
			t.Error("proxy port should not be present when proxy is disabled")
		}
	}
	for _, v := range ds.Spec.Template.Spec.Volumes {
		if v.Name == pkgcommon.ClusterGuardProxyTLSVolumeName {
			t.Error("proxy-tls volume should not be present when proxy is disabled")
		}
	}
	for _, vm := range ds.Spec.Template.Spec.Containers[0].VolumeMounts {
		if vm.MountPath == pkgcommon.ClusterGuardProxyMountPath {
			t.Error("proxy volumemount should not be present when proxy is disabled")
		}
	}
}

func TestDaemonSetProxyEnabledAddsPortVolumeMount(t *testing.T) {
	enabled := true
	port := int32(48080)
	n := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
			Image:            "quay.io/crowdstrike/falcon-sensor:latest",
		},
		NodeSensor: falconv1alpha1.FalconClusterGuardNodeSpec{
			Guardian: falconv1alpha1.FalconClusterGuardGuardian{
				Proxy: falconv1alpha1.FalconClusterGuardGuardianProxy{
					Enabled: &enabled,
					Port:    &port,
				},
			},
		},
	})
	ds := n.daemonSet()

	// Container port
	foundPort := false
	for _, p := range ds.Spec.Template.Spec.Containers[0].Ports {
		if p.Name == pkgcommon.ClusterGuardProxyPortName {
			foundPort = true
			if p.ContainerPort != port {
				t.Errorf("expected containerPort %d, got %d", port, p.ContainerPort)
			}
			if p.HostPort != port {
				t.Errorf("expected hostPort %d to match the API server default for hostNetwork pods, got %d", port, p.HostPort)
			}
			if p.Protocol != corev1.ProtocolTCP {
				t.Errorf("expected protocol TCP, got %q", p.Protocol)
			}
		}
	}
	if !foundPort {
		t.Error("expected proxy container port to be present when proxy is enabled")
	}

	// Volume with optional: true
	foundVol := false
	for _, v := range ds.Spec.Template.Spec.Volumes {
		if v.Name == pkgcommon.ClusterGuardProxyTLSVolumeName {
			foundVol = true
			if v.VolumeSource.Secret == nil {
				t.Fatal("expected secret volume source for proxy-tls")
			}
			if v.VolumeSource.Secret.Optional == nil || !*v.VolumeSource.Secret.Optional {
				t.Error("expected proxy-tls volume to have Optional=true")
			}
		}
	}
	if !foundVol {
		t.Error("expected proxy-tls volume to be present when proxy is enabled")
	}

	// VolumeMount
	foundMount := false
	for _, vm := range ds.Spec.Template.Spec.Containers[0].VolumeMounts {
		if vm.Name == pkgcommon.ClusterGuardProxyTLSVolumeName {
			foundMount = true
			if vm.MountPath != pkgcommon.ClusterGuardProxyMountPath {
				t.Errorf("expected mountPath %q, got %q", pkgcommon.ClusterGuardProxyMountPath, vm.MountPath)
			}
			if !vm.ReadOnly {
				t.Error("expected proxy volumemount to be ReadOnly")
			}
		}
	}
	if !foundMount {
		t.Error("expected proxy volumemount to be present when proxy is enabled")
	}
}

func TestDaemonSetProxyEnabledCustomTLSSecretName(t *testing.T) {
	enabled := true
	port := int32(48080)
	n := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
			Image:            "quay.io/crowdstrike/falcon-sensor:latest",
		},
		NodeSensor: falconv1alpha1.FalconClusterGuardNodeSpec{
			Guardian: falconv1alpha1.FalconClusterGuardGuardian{
				Proxy: falconv1alpha1.FalconClusterGuardGuardianProxy{
					Enabled:       &enabled,
					Port:          &port,
					TLSSecretName: "my-custom-tls",
				},
			},
		},
	})
	ds := n.daemonSet()

	for _, v := range ds.Spec.Template.Spec.Volumes {
		if v.Name == pkgcommon.ClusterGuardProxyTLSVolumeName {
			if v.VolumeSource.Secret.SecretName != "my-custom-tls" {
				t.Errorf("expected secret name %q, got %q", "my-custom-tls", v.VolumeSource.Secret.SecretName)
			}
			return
		}
	}
	t.Error("proxy-tls volume not found")
}

// DNSConfig builder tests

func TestDaemonSetNilDNSConfigNotApplied(t *testing.T) {
	n := New(nil, Config{BaseConfig: components.BaseConfig{
		InstallNamespace: "falcon-clusterguard",
		Image:            "quay.io/crowdstrike/falcon-sensor:latest",
	}})
	ds := n.daemonSet()

	if ds.Spec.Template.Spec.DNSConfig != nil {
		t.Error("expected nil DNSConfig when not configured")
	}
}

func TestNodeSensorResourcesHaveHelmAppLabels(t *testing.T) {
	n := New(nil, Config{BaseConfig: components.BaseConfig{
		Request:          ctrl.Request{NamespacedName: types.NamespacedName{Name: "my-fcg"}},
		InstallNamespace: "falcon-clusterguard",
	}})
	ds := n.daemonSet()
	cleanup := n.cleanupDaemonSet()

	tests := []struct {
		name    string
		labels  map[string]string
		appName string
	}{
		{"sensor DaemonSet", ds.Labels, pkgcommon.ClusterGuardComponentName},
		{"sensor pod template", ds.Spec.Template.Labels, pkgcommon.ClusterGuardComponentName},
		{"cleanup DaemonSet", cleanup.Labels, pkgcommon.ClusterGuardComponentName + "-node-cleanup"},
		{"cleanup pod template", cleanup.Spec.Template.Labels, pkgcommon.ClusterGuardComponentName + "-node-cleanup"},
		{"ConfigMap", n.configMap().Labels, pkgcommon.ClusterGuardComponentName},
		{"ServiceAccount", n.serviceAccount().Labels, pkgcommon.ClusterGuardComponentName},
		{"cleanup ServiceAccount", n.cleanupServiceAccount().Labels, pkgcommon.ClusterGuardComponentName},
		{"ClusterRoleBinding", n.clusterRoleBinding().Labels, pkgcommon.ClusterGuardComponentName},
		{"proxy Service", n.proxyService().Labels, pkgcommon.ClusterGuardComponentName},
	}
	for _, tt := range tests {
		want := map[string]string{
			pkgcommon.KubernetesNameKey:      tt.appName,
			pkgcommon.KubernetesInstanceKey:  "my-fcg",
			pkgcommon.KubernetesManagedByKey: "falcon-operator",
			pkgcommon.KubernetesComponentKey: pkgcommon.FalconKernelSensor,
		}
		for k, v := range want {
			if got := tt.labels[k]; got != v {
				t.Errorf("%s: expected %s=%q, got %q", tt.name, k, v, got)
			}
		}
	}
}
