package admission

import (
	"testing"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/internal/controller/components"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
)

func TestClusterGuardDeploymentReturnsDeployment(t *testing.T) {
	// Default deployment name is the hardcoded const when NamePrefix is empty
	name := common.AdmissionDeploymentName
	namespace := "falcon-clusterguard"
	imageUri := "quay.io/crowdstrike/falcon-clusterguard:latest"
	imagePullPolicy := corev1.PullIfNotPresent
	imagePullSecrets := []corev1.LocalObjectReference{{Name: "mysecret"}}

	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: namespace,
			Image:            imageUri,
			ImagePullPolicy:  imagePullPolicy,
			ImagePullSecrets: imagePullSecrets,
		},
	})
	dep := a.Deployment()

	if dep == nil {
		t.Fatal("expected non-nil Deployment")
	}
	if dep.Name != name {
		t.Errorf("expected name %q, got %q", name, dep.Name)
	}
	if dep.Namespace != namespace {
		t.Errorf("expected namespace %q, got %q", namespace, dep.Namespace)
	}
	if len(dep.Spec.Template.Spec.Containers) != 3 {
		t.Errorf("expected 3 containers, got %d", len(dep.Spec.Template.Spec.Containers))
	}
	if dep.Spec.Template.Spec.Containers[0].Image != imageUri {
		t.Errorf("expected image %q, got %q", imageUri, dep.Spec.Template.Spec.Containers[0].Image)
	}
}

func TestClusterGuardDeploymentUsesNamePrefix(t *testing.T) {
	prefix := "my-custom-guard"
	namespace := "test-ns"
	imageUri := "quay.io/crowdstrike/falcon-clusterguard:latest"

	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: namespace,
			Image:            imageUri,
			NamePrefix:       prefix,
		},
	})
	dep := a.Deployment()

	if dep == nil {
		t.Fatal("expected non-nil Deployment")
	}
	if dep.Name != common.AdmissionDeploymentName {
		t.Errorf("expected deployment name %q, got %q", common.AdmissionDeploymentName, dep.Name)
	}
	// SA name should be derived from prefix
	expectedSA := prefix + "-sa"
	if dep.Spec.Template.Spec.ServiceAccountName != expectedSA {
		t.Errorf("expected ServiceAccountName %q, got %q", expectedSA, dep.Spec.Template.Spec.ServiceAccountName)
	}
	// TLS volume should reference prefix-derived secret
	expectedTLSSecretName := prefix + "-tls"
	foundTLS := false
	for _, v := range dep.Spec.Template.Spec.Volumes {
		if v.VolumeSource.Secret != nil && v.VolumeSource.Secret.SecretName == expectedTLSSecretName {
			foundTLS = true
		}
	}
	if !foundTLS {
		t.Errorf("expected TLS secret name %q in volumes, not found", expectedTLSSecretName)
	}
	// ConfigMap reference should use prefix-derived name
	expectedCM := prefix + "-config"
	foundCM := false
	for _, c := range dep.Spec.Template.Spec.Containers {
		for _, ef := range c.EnvFrom {
			if ef.ConfigMapRef != nil && ef.ConfigMapRef.Name == expectedCM {
				foundCM = true
			}
		}
	}
	if !foundCM {
		t.Errorf("expected ConfigMap name %q in container envFrom, not found", expectedCM)
	}
}

func TestClusterGuardValidatingWebhookReturnsWebhook(t *testing.T) {
	namespace := "falcon-clusterguard"
	caBundle := []byte("fake-ca")
	extraDisabledNamespaces := []string{"kube-system"}

	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: namespace,
		},
		AdmissionConfig: falconv1alpha1.FalconClusterGuardAdmissionSpec{
			DisabledNamespaces: falconv1alpha1.FalconClusterGuardAdmissionNamespace{
				Namespaces: extraDisabledNamespaces,
			},
		},
	})
	webhook := a.ValidatingWebhook(caBundle)

	if webhook == nil {
		t.Fatal("expected non-nil ValidatingWebhookConfiguration")
	}
	if webhook.Name != common.AdmissionValidatingWebhookName {
		t.Errorf("expected name %q, got %q", common.AdmissionValidatingWebhookName, webhook.Name)
	}
	if len(webhook.Webhooks) != 3 {
		t.Errorf("expected 3 webhooks, got %d", len(webhook.Webhooks))
	}
}

func TestClusterGuardValidatingWebhookDeduplicatesNamespaces(t *testing.T) {
	namespace := "falcon-clusterguard"
	caBundle := []byte("fake-ca")
	// Pass a duplicate of a default disabled namespace
	extraDisabledNamespaces := []string{namespace, namespace}

	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: namespace,
		},
		AdmissionConfig: falconv1alpha1.FalconClusterGuardAdmissionSpec{
			DisabledNamespaces: falconv1alpha1.FalconClusterGuardAdmissionNamespace{
				Namespaces: extraDisabledNamespaces,
			},
		},
	})
	webhook := a.ValidatingWebhook(caBundle)

	if webhook == nil {
		t.Fatal("expected non-nil ValidatingWebhookConfiguration")
	}
	// The main webhook's namespace selector values should not have duplicates
	nsSelector := webhook.Webhooks[0].NamespaceSelector
	if nsSelector == nil || len(nsSelector.MatchExpressions) == 0 {
		t.Fatal("expected namespace selector with match expressions")
	}
	values := nsSelector.MatchExpressions[0].Values
	seen := map[string]int{}
	for _, v := range values {
		seen[v]++
		if seen[v] > 1 {
			t.Errorf("duplicate namespace %q found in selector values", v)
		}
	}
}

// configMap builder tests

func TestAdmissionConfigMapHasRequiredKeys(t *testing.T) {
	trueVal := true
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
			Cid:              "abc123",
		},
		AdmissionConfig: falconv1alpha1.FalconClusterGuardAdmissionSpec{
			AdmissionControlEnabled: &trueVal,
		},
	})
	cm := a.configMap()

	if cm == nil {
		t.Fatal("expected non-nil ConfigMap")
	}

	requiredKeys := []string{
		"FALCON_MODE",
		"WEBHOOK_PORT",
		"GRPC_PORT",
		"WATCHER_HTTP_PORT",
		"__CS_ADMISSION_CONTROL_ENABLED",
		"__CS_WATCH_EVENTS_ENABLED",
		"__CS_SNAPSHOTS_ENABLED",
		"__CS_SNAPSHOT_INTERVAL",
		"FALCONCTL_OPT_CID",
	}
	for _, k := range requiredKeys {
		if _, ok := cm.Data[k]; !ok {
			t.Errorf("expected ConfigMap key %q to be present", k)
		}
	}

	if cm.Data["FALCON_MODE"] != "kac" {
		t.Errorf("expected FALCON_MODE=kac, got %q", cm.Data["FALCON_MODE"])
	}
}

func TestAdmissionConfigMapPortValues(t *testing.T) {
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
		},
	})
	cm := a.configMap()

	if cm.Data["WEBHOOK_PORT"] != common.AdmissionWebhookPortStr {
		t.Errorf("expected WEBHOOK_PORT=%q, got %q", common.AdmissionWebhookPortStr, cm.Data["WEBHOOK_PORT"])
	}
	if cm.Data["GRPC_PORT"] != common.AdmissionGRPCPortStr {
		t.Errorf("expected GRPC_PORT=%q, got %q", common.AdmissionGRPCPortStr, cm.Data["GRPC_PORT"])
	}
	if cm.Data["WATCHER_HTTP_PORT"] != common.AdmissionWatcherHTTPPortStr {
		t.Errorf("expected WATCHER_HTTP_PORT=%q, got %q", common.AdmissionWatcherHTTPPortStr, cm.Data["WATCHER_HTTP_PORT"])
	}
}

func TestAdmissionConfigMapAdmissionControlEnabled(t *testing.T) {
	trueVal := true
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
		},
		AdmissionConfig: falconv1alpha1.FalconClusterGuardAdmissionSpec{AdmissionControlEnabled: &trueVal},
	})
	cm := a.configMap()

	if cm.Data["__CS_ADMISSION_CONTROL_ENABLED"] != "true" {
		t.Errorf("expected __CS_ADMISSION_CONTROL_ENABLED=true, got %q", cm.Data["__CS_ADMISSION_CONTROL_ENABLED"])
	}
}

func TestAdmissionConfigMapAdmissionControlDisabled(t *testing.T) {
	falseVal := false
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
		},
		AdmissionConfig: falconv1alpha1.FalconClusterGuardAdmissionSpec{AdmissionControlEnabled: &falseVal},
	})
	cm := a.configMap()

	if cm.Data["__CS_ADMISSION_CONTROL_ENABLED"] != "false" {
		t.Errorf("expected __CS_ADMISSION_CONTROL_ENABLED=false, got %q", cm.Data["__CS_ADMISSION_CONTROL_ENABLED"])
	}
}

func TestAdmissionConfigMapAdmissionControlNilIsTrue(t *testing.T) {
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
		},
		AdmissionConfig: falconv1alpha1.FalconClusterGuardAdmissionSpec{AdmissionControlEnabled: nil},
	})
	cm := a.configMap()

	if cm.Data["__CS_ADMISSION_CONTROL_ENABLED"] != "true" {
		t.Errorf("expected __CS_ADMISSION_CONTROL_ENABLED=true when nil (default enabled), got %q", cm.Data["__CS_ADMISSION_CONTROL_ENABLED"])
	}
}

func TestAdmissionConfigMapCidIsSet(t *testing.T) {
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
			Cid:              "asdfasdf00000000000000000000000000000000-ab",
		},
	})
	cm := a.configMap()

	if cm.Data["FALCONCTL_OPT_CID"] != "asdfasdf00000000000000000000000000000000-ab" {
		t.Errorf("expected FALCONCTL_OPT_CID to match, got %q", cm.Data["FALCONCTL_OPT_CID"])
	}
}

func TestAdmissionConfigMapNameUsesPrefix(t *testing.T) {
	prefix := "my-guard"
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "test-ns",
			NamePrefix:       prefix,
		},
	})
	cm := a.configMap()

	expected := prefix + "-config"
	if cm.Name != expected {
		t.Errorf("expected ConfigMap name %q, got %q", expected, cm.Name)
	}
	if cm.Namespace != "test-ns" {
		t.Errorf("expected namespace %q, got %q", "test-ns", cm.Namespace)
	}
}

func TestAdmissionConfigMapDefaultName(t *testing.T) {
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
		},
	})
	cm := a.configMap()

	if cm.Name != "falcon-cluster-sensor-config" {
		t.Errorf("expected default ConfigMap name %q, got %q", "falcon-cluster-sensor-config", cm.Name)
	}
}

func TestClusterNameConfigMapHasClusterName(t *testing.T) {
	clusterName := "my-cluster"
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
		},
		ClusterName: &clusterName,
	})
	cm := a.clusterNameConfigMap()

	if cm == nil {
		t.Fatal("expected non-nil ClusterName ConfigMap")
	}
	if cm.Data["ClusterName"] != clusterName {
		t.Errorf("expected ClusterName=%q, got %q", clusterName, cm.Data["ClusterName"])
	}
}

func TestClusterNameConfigMapName(t *testing.T) {
	clusterName := "my-cluster"
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
		},
		ClusterName: &clusterName,
	})
	cm := a.clusterNameConfigMap()

	if cm.Name != common.FalconAdmissionClusterNameConfigMapName {
		t.Errorf("expected name %q, got %q", common.FalconAdmissionClusterNameConfigMapName, cm.Name)
	}
}

// ServiceAccount builder tests

func TestAdmissionServiceAccountName(t *testing.T) {
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
		},
	})
	sa := a.serviceAccount()

	if sa == nil {
		t.Fatal("expected non-nil ServiceAccount")
	}
	if sa.Name != "falcon-cluster-sensor-sa" {
		t.Errorf("expected name %q, got %q", "falcon-cluster-sensor-sa", sa.Name)
	}
	if sa.Namespace != "falcon-clusterguard" {
		t.Errorf("expected namespace %q, got %q", "falcon-clusterguard", sa.Namespace)
	}
}

func TestAdmissionServiceAccountNameUsesPrefix(t *testing.T) {
	prefix := "my-guard"
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "test-ns",
			NamePrefix:       prefix,
		},
	})
	sa := a.serviceAccount()

	if sa.Name != prefix+"-sa" {
		t.Errorf("expected name %q, got %q", prefix+"-sa", sa.Name)
	}
	if sa.Namespace != "test-ns" {
		t.Errorf("expected namespace %q, got %q", "test-ns", sa.Namespace)
	}
}

func TestAdmissionServiceAccountImagePullSecrets(t *testing.T) {
	secrets := []corev1.LocalObjectReference{{Name: "my-pull-secret"}, {Name: "other-secret"}}
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
			ImagePullSecrets: secrets,
		},
	})
	sa := a.serviceAccount()

	if len(sa.ImagePullSecrets) != 2 {
		t.Fatalf("expected 2 ImagePullSecrets, got %d", len(sa.ImagePullSecrets))
	}
	if sa.ImagePullSecrets[0].Name != "my-pull-secret" {
		t.Errorf("expected first secret %q, got %q", "my-pull-secret", sa.ImagePullSecrets[0].Name)
	}
}

func TestAdmissionServiceAccountAnnotations(t *testing.T) {
	annotations := map[string]string{"iam.amazonaws.com/role": "arn:aws:iam::123:role/my-role"}
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
		},
		AdmissionConfig: falconv1alpha1.FalconClusterGuardAdmissionSpec{
			ServiceAccount: falconv1alpha1.FalconClusterGuardAdmissionServiceAccount{Annotations: annotations},
		},
	})
	sa := a.serviceAccount()

	if sa.Annotations["iam.amazonaws.com/role"] != "arn:aws:iam::123:role/my-role" {
		t.Errorf("expected annotation to be propagated, got %q", sa.Annotations["iam.amazonaws.com/role"])
	}
}

// ClusterRoleBinding builder tests

func TestAdmissionClusterRoleBindingName(t *testing.T) {
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
		},
	})
	crb := a.clusterRoleBinding()

	if crb == nil {
		t.Fatal("expected non-nil ClusterRoleBinding")
	}
	if crb.Name != "falcon-cluster-sensor-security-crb" {
		t.Errorf("expected name %q, got %q", "falcon-cluster-sensor-security-crb", crb.Name)
	}
}

func TestAdmissionClusterRoleBindingNameUsesPrefix(t *testing.T) {
	prefix := "my-guard"
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "test-ns",
			NamePrefix:       prefix,
		},
	})
	crb := a.clusterRoleBinding()

	if crb.Name != prefix+"-security-crb" {
		t.Errorf("expected name %q, got %q", prefix+"-security-crb", crb.Name)
	}
}

func TestAdmissionClusterRoleBindingRoleRef(t *testing.T) {
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
		},
	})
	crb := a.clusterRoleBinding()

	if crb.RoleRef.Kind != "ClusterRole" {
		t.Errorf("expected RoleRef.Kind=ClusterRole, got %q", crb.RoleRef.Kind)
	}
	if crb.RoleRef.Name != common.FCGAdmissionClusterRoleName {
		t.Errorf("expected RoleRef.Name=%q, got %q", common.FCGAdmissionClusterRoleName, crb.RoleRef.Name)
	}
	if crb.RoleRef.APIGroup != "rbac.authorization.k8s.io" {
		t.Errorf("expected RoleRef.APIGroup=rbac.authorization.k8s.io, got %q", crb.RoleRef.APIGroup)
	}
}

func TestAdmissionClusterRoleBindingSubjectPointsToSA(t *testing.T) {
	prefix := "my-guard"
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "test-ns",
			NamePrefix:       prefix,
		},
	})
	crb := a.clusterRoleBinding()

	if len(crb.Subjects) != 1 {
		t.Fatalf("expected 1 subject, got %d", len(crb.Subjects))
	}
	subj := crb.Subjects[0]
	if subj.Kind != "ServiceAccount" {
		t.Errorf("expected subject Kind=ServiceAccount, got %q", subj.Kind)
	}
	if subj.Name != prefix+"-sa" {
		t.Errorf("expected subject Name=%q, got %q", prefix+"-sa", subj.Name)
	}
	if subj.Namespace != "test-ns" {
		t.Errorf("expected subject Namespace=%q, got %q", "test-ns", subj.Namespace)
	}
}

// RoleBinding builder tests

func TestAdmissionRoleBindingName(t *testing.T) {
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
		},
	})
	rb := a.roleBinding()

	if rb == nil {
		t.Fatal("expected non-nil RoleBinding")
	}
	if rb.Name != "falcon-cluster-sensor-rolebinding" {
		t.Errorf("expected name %q, got %q", "falcon-cluster-sensor-rolebinding", rb.Name)
	}
	if rb.Namespace != "falcon-clusterguard" {
		t.Errorf("expected namespace %q, got %q", "falcon-clusterguard", rb.Namespace)
	}
}

func TestAdmissionRoleBindingNameUsesPrefix(t *testing.T) {
	prefix := "my-guard"
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "test-ns",
			NamePrefix:       prefix,
		},
	})
	rb := a.roleBinding()

	if rb.Name != prefix+"-rolebinding" {
		t.Errorf("expected name %q, got %q", prefix+"-rolebinding", rb.Name)
	}
}

func TestAdmissionRoleBindingRoleRef(t *testing.T) {
	prefix := "my-guard"
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "test-ns",
			NamePrefix:       prefix,
		},
	})
	rb := a.roleBinding()

	if rb.RoleRef.Kind != "Role" {
		t.Errorf("expected RoleRef.Kind=Role, got %q", rb.RoleRef.Kind)
	}
	if rb.RoleRef.Name != prefix+"-namespace-role" {
		t.Errorf("expected RoleRef.Name=%q, got %q", prefix+"-namespace-role", rb.RoleRef.Name)
	}
}

func TestAdmissionRoleBindingSubjectPointsToSA(t *testing.T) {
	prefix := "my-guard"
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "test-ns",
			NamePrefix:       prefix,
		},
	})
	rb := a.roleBinding()

	if len(rb.Subjects) != 1 {
		t.Fatalf("expected 1 subject, got %d", len(rb.Subjects))
	}
	if rb.Subjects[0].Name != prefix+"-sa" {
		t.Errorf("expected subject Name=%q, got %q", prefix+"-sa", rb.Subjects[0].Name)
	}
	if rb.Subjects[0].Namespace != "test-ns" {
		t.Errorf("expected subject Namespace=%q, got %q", "test-ns", rb.Subjects[0].Namespace)
	}
}

// Role builder tests

func TestAdmissionRoleName(t *testing.T) {
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
		},
	})
	role := a.role()

	if role == nil {
		t.Fatal("expected non-nil Role")
	}
	if role.Name != "falcon-cluster-sensor-namespace-role" {
		t.Errorf("expected name %q, got %q", "falcon-cluster-sensor-namespace-role", role.Name)
	}
	if role.Namespace != "falcon-clusterguard" {
		t.Errorf("expected namespace %q, got %q", "falcon-clusterguard", role.Namespace)
	}
}

func TestAdmissionRoleNameUsesPrefix(t *testing.T) {
	prefix := "my-guard"
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "test-ns",
			NamePrefix:       prefix,
		},
	})
	role := a.role()

	if role.Name != prefix+"-namespace-role" {
		t.Errorf("expected name %q, got %q", prefix+"-namespace-role", role.Name)
	}
}

func TestAdmissionRoleHasPolicyRules(t *testing.T) {
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
		},
	})
	role := a.role()

	if len(role.Rules) == 0 {
		t.Fatal("expected Role to have policy rules")
	}

	// Verify configmaps rule exists
	foundConfigMaps := false
	foundPods := false
	for _, rule := range role.Rules {
		for _, res := range rule.Resources {
			if res == "configmaps" {
				foundConfigMaps = true
			}
			if res == "pods" {
				foundPods = true
			}
		}
	}
	if !foundConfigMaps {
		t.Error("expected Role to have a rule for configmaps")
	}
	if !foundPods {
		t.Error("expected Role to have a rule for pods")
	}
}

// WebhookService builder tests

func TestAdmissionWebhookServiceName(t *testing.T) {
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
		},
	})
	svc := a.webhookService()

	if svc == nil {
		t.Fatal("expected non-nil Service")
	}
	if svc.Name != common.AdmissionWebhookServiceName {
		t.Errorf("expected name %q, got %q", common.AdmissionWebhookServiceName, svc.Name)
	}
	if svc.Namespace != "falcon-clusterguard" {
		t.Errorf("expected namespace %q, got %q", "falcon-clusterguard", svc.Namespace)
	}
}

func TestAdmissionWebhookServiceSelectorUsesFalconKAC(t *testing.T) {
	prefix := "my-guard"
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "test-ns",
			NamePrefix:       prefix,
		},
	})
	svc := a.webhookService()

	if svc.Spec.Selector["app"] != common.AdmissionServiceApp {
		t.Errorf("expected selector app=%q, got %q", common.AdmissionServiceApp, svc.Spec.Selector["app"])
	}
}

func TestAdmissionWebhookServicePort(t *testing.T) {
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
		},
	})
	svc := a.webhookService()

	if len(svc.Spec.Ports) != 1 {
		t.Fatalf("expected 1 port, got %d", len(svc.Spec.Ports))
	}
	if svc.Spec.Ports[0].Port != common.FalconServiceHTTPSPort {
		t.Errorf("expected port %d, got %d", common.FalconServiceHTTPSPort, svc.Spec.Ports[0].Port)
	}
}

// APIService builder tests

func TestAdmissionAPIServiceName(t *testing.T) {
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
		},
	})
	svc := a.apiService()

	if svc == nil {
		t.Fatal("expected non-nil Service")
	}
	if svc.Name != common.AdmissionAPIServiceName {
		t.Errorf("expected name %q, got %q", common.AdmissionAPIServiceName, svc.Name)
	}
	if svc.Namespace != "falcon-clusterguard" {
		t.Errorf("expected namespace %q, got %q", "falcon-clusterguard", svc.Namespace)
	}
}

func TestAdmissionAPIServiceSelectorUsesFalconKAC(t *testing.T) {
	prefix := "my-guard"
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "test-ns",
			NamePrefix:       prefix,
		},
	})
	svc := a.apiService()

	if svc.Spec.Selector["app"] != common.AdmissionServiceApp {
		t.Errorf("expected selector app=%q, got %q", common.AdmissionServiceApp, svc.Spec.Selector["app"])
	}
}

func TestAdmissionAPIServicePort(t *testing.T) {
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
		},
	})
	svc := a.apiService()

	if len(svc.Spec.Ports) != 1 {
		t.Fatalf("expected 1 port, got %d", len(svc.Spec.Ports))
	}
	if svc.Spec.Ports[0].Port != common.FalconServiceHTTPSPort {
		t.Errorf("expected port %d, got %d", common.FalconServiceHTTPSPort, svc.Spec.Ports[0].Port)
	}
}

func TestAdmissionWebhookAndAPIServiceHaveDifferentPortNames(t *testing.T) {
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
		},
	})
	webhook := a.webhookService()
	api := a.apiService()

	if webhook.Spec.Ports[0].Name == api.Spec.Ports[0].Name {
		t.Errorf("expected webhook and API service port names to differ, both are %q",
			webhook.Spec.Ports[0].Name)
	}
}
