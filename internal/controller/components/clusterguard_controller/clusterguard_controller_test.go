package clusterguard_controller

import (
	"testing"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/internal/controller/components"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
)

func TestClusterGuardDeploymentReturnsDeployment(t *testing.T) {
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

func TestClusterGuardValidatingWebhookReturnsWebhook(t *testing.T) {
	namespace := "falcon-clusterguard"
	caBundle := []byte("fake-ca")
	extraDisabledNamespaces := []string{"kube-system"}

	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: namespace,
		},
		ClusterGuardControllerConfig: falconv1alpha1.FalconClusterGuardController{
			DisabledNamespaces: falconv1alpha1.FalconClusterGuardControllerNamespace{
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
		ClusterGuardControllerConfig: falconv1alpha1.FalconClusterGuardController{
			DisabledNamespaces: falconv1alpha1.FalconClusterGuardControllerNamespace{
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
		ClusterGuardControllerConfig: falconv1alpha1.FalconClusterGuardController{
			AdmissionControlEnabled: &trueVal,
		},
	})
	cm := a.configMap()

	if cm == nil {
		t.Fatal("expected non-nil ConfigMap")
	}

	requiredKeys := []string{
		"FALCON_MODE",
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

func TestAdmissionConfigMapAdmissionControlEnabled(t *testing.T) {
	trueVal := true
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
		},
		ClusterGuardControllerConfig: falconv1alpha1.FalconClusterGuardController{AdmissionControlEnabled: &trueVal},
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
		ClusterGuardControllerConfig: falconv1alpha1.FalconClusterGuardController{AdmissionControlEnabled: &falseVal},
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
		ClusterGuardControllerConfig: falconv1alpha1.FalconClusterGuardController{AdmissionControlEnabled: nil},
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

func TestAdmissionConfigMapDefaultName(t *testing.T) {
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
		},
	})
	cm := a.configMap()

	if cm.Name != "falcon-clusterguard-config" {
		t.Errorf("expected default ConfigMap name %q, got %q", "falcon-clusterguard-config", cm.Name)
	}
}

func TestAdmissionConfigMapTLSVersionMinimumWhenSet(t *testing.T) {
	tlsMin := "TLS1.3"
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
		},
		ClusterGuardControllerConfig: falconv1alpha1.FalconClusterGuardController{
			TLSVersionMinimum: &tlsMin,
		},
	})
	cm := a.configMap()

	if cm.Data["__CS_TLS_PROTOCOL_MIN"] != "TLS1.3" {
		t.Errorf("expected __CS_TLS_PROTOCOL_MIN=TLS1.3, got %q", cm.Data["__CS_TLS_PROTOCOL_MIN"])
	}
}

func TestAdmissionConfigMapTLSVersionMinimumAbsentWhenUnset(t *testing.T) {
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
		},
	})
	cm := a.configMap()

	if _, ok := cm.Data["__CS_TLS_PROTOCOL_MIN"]; ok {
		t.Errorf("expected __CS_TLS_PROTOCOL_MIN to be absent when TLSVersionMinimum is nil, got %q", cm.Data["__CS_TLS_PROTOCOL_MIN"])
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
	if sa.Name != "falcon-cg-controller-sa" {
		t.Errorf("expected name %q, got %q", "falcon-cg-controller-sa", sa.Name)
	}
	if sa.Namespace != "falcon-clusterguard" {
		t.Errorf("expected namespace %q, got %q", "falcon-clusterguard", sa.Namespace)
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
		ClusterGuardControllerConfig: falconv1alpha1.FalconClusterGuardController{
			ServiceAccount: falconv1alpha1.FalconClusterGuardControllerServiceAccount{Annotations: annotations},
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
	if crb.Name != "falcon-clusterguard-security-crb" {
		t.Errorf("expected name %q, got %q", "falcon-clusterguard-security-crb", crb.Name)
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
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "test-ns",
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
	if subj.Name != common.AdmissionModuleServiceAccountName {
		t.Errorf("expected subject Name=%q, got %q", common.AdmissionModuleServiceAccountName, subj.Name)
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
	if rb.Name != "falcon-clusterguard-rolebinding" {
		t.Errorf("expected name %q, got %q", "falcon-clusterguard-rolebinding", rb.Name)
	}
	if rb.Namespace != "falcon-clusterguard" {
		t.Errorf("expected namespace %q, got %q", "falcon-clusterguard", rb.Namespace)
	}
}

func TestAdmissionRoleBindingRoleRef(t *testing.T) {
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "test-ns",
		},
	})
	rb := a.roleBinding()

	if rb.RoleRef.Kind != "Role" {
		t.Errorf("expected RoleRef.Kind=Role, got %q", rb.RoleRef.Kind)
	}
	if rb.RoleRef.Name != common.AdmissionNamespaceRoleName {
		t.Errorf("expected RoleRef.Name=%q, got %q", common.AdmissionNamespaceRoleName, rb.RoleRef.Name)
	}
}

func TestAdmissionRoleBindingSubjectPointsToSA(t *testing.T) {
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "test-ns",
		},
	})
	rb := a.roleBinding()

	if len(rb.Subjects) != 1 {
		t.Fatalf("expected 1 subject, got %d", len(rb.Subjects))
	}
	if rb.Subjects[0].Name != common.AdmissionModuleServiceAccountName {
		t.Errorf("expected subject Name=%q, got %q", common.AdmissionModuleServiceAccountName, rb.Subjects[0].Name)
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
	if role.Name != "falcon-clusterguard-namespace-role" {
		t.Errorf("expected name %q, got %q", "falcon-clusterguard-namespace-role", role.Name)
	}
	if role.Namespace != "falcon-clusterguard" {
		t.Errorf("expected namespace %q, got %q", "falcon-clusterguard", role.Namespace)
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
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "test-ns",
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
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "test-ns",
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

// ResourceQuota builder tests

func TestResourceQuotaDefaultPods(t *testing.T) {
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{InstallNamespace: "falcon-clusterguard"},
	})
	rq := a.resourceQuota()

	pods, ok := rq.Spec.Hard[corev1.ResourcePods]
	if !ok {
		t.Fatal("expected ResourcePods in Hard limits")
	}
	if pods.Value() != int64(defaultResourceQuotaPods) {
		t.Errorf("expected %d pods, got %d", defaultResourceQuotaPods, pods.Value())
	}
}

func TestResourceQuotaCustomPods(t *testing.T) {
	custom := int32(5)
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{InstallNamespace: "falcon-clusterguard"},
		ClusterGuardControllerConfig: falconv1alpha1.FalconClusterGuardController{
			ResourceQuotaPods: &custom,
		},
	})
	rq := a.resourceQuota()

	pods, ok := rq.Spec.Hard[corev1.ResourcePods]
	if !ok {
		t.Fatal("expected ResourcePods in Hard limits")
	}
	if pods.Value() != int64(custom) {
		t.Errorf("expected %d pods, got %d", custom, pods.Value())
	}
}

func TestResourceQuotaHasPriorityClassScopeSelector(t *testing.T) {
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{InstallNamespace: "falcon-clusterguard"},
	})
	rq := a.resourceQuota()

	if rq.Spec.ScopeSelector == nil {
		t.Fatal("expected non-nil ScopeSelector")
	}
	if len(rq.Spec.ScopeSelector.MatchExpressions) != 1 {
		t.Fatalf("expected 1 scope expression, got %d", len(rq.Spec.ScopeSelector.MatchExpressions))
	}
	expr := rq.Spec.ScopeSelector.MatchExpressions[0]
	if expr.ScopeName != corev1.ResourceQuotaScopePriorityClass {
		t.Errorf("expected ScopeName %q, got %q", corev1.ResourceQuotaScopePriorityClass, expr.ScopeName)
	}
}
