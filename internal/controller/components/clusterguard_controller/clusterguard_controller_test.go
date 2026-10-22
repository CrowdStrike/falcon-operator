package clusterguard_controller

import (
	"slices"
	"testing"

	falconv1alpha1 "github.com/crowdstrike/falcon-operator/api/falcon/v1alpha1"
	"github.com/crowdstrike/falcon-operator/internal/controller/components"
	"github.com/crowdstrike/falcon-operator/pkg/common"
	corev1 "k8s.io/api/core/v1"
)

func TestClusterGuardDeploymentReturnsDeployment(t *testing.T) {
	name := common.ClusterGuardControllerDeploymentName
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
	// Container names and the app label match the falcon-clusterguard Helm chart.
	var names []string
	for _, c := range dep.Spec.Template.Spec.Containers {
		names = append(names, c.Name)
	}
	if want := []string{"falcon-ac", "falcon-client", "falcon-watcher"}; !slices.Equal(names, want) {
		t.Errorf("expected containers %v, got %v", want, names)
	}
	if got := dep.Spec.Template.Labels["app"]; got != "falcon-clusterguard" {
		t.Errorf("expected pod label app=falcon-clusterguard, got %q", got)
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
	if webhook.Name != common.ClusterGuardControllerValidatingWebhookName {
		t.Errorf("expected name %q, got %q", common.ClusterGuardControllerValidatingWebhookName, webhook.Name)
	}
	if len(webhook.Webhooks) != 2 {
		t.Errorf("expected 2 webhooks, got %d", len(webhook.Webhooks))
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

func TestAdmissionConfigMapIARNamespaceIsInstallNamespace(t *testing.T) {
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
		},
	})
	cm := a.configMap()

	if cm.Data["__CS_IAR_NAMESPACE"] != "falcon-clusterguard" {
		t.Errorf("expected __CS_IAR_NAMESPACE=falcon-clusterguard, got %q", cm.Data["__CS_IAR_NAMESPACE"])
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

// ServiceAccount builder tests

// assets.ServiceAccount is covered in the assets package; this only checks which
// config fields the controller feeds into it.
func TestAdmissionServiceAccountWiring(t *testing.T) {
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "falcon-clusterguard",
			ImagePullSecrets: []corev1.LocalObjectReference{{Name: "my-pull-secret"}},
		},
		ClusterGuardControllerConfig: falconv1alpha1.FalconClusterGuardController{
			ServiceAccount: falconv1alpha1.FalconClusterGuardControllerServiceAccount{
				Annotations: map[string]string{"iam.amazonaws.com/role": "arn:aws:iam::123:role/my-role"},
			},
		},
	})
	sa := a.serviceAccount()

	if len(sa.ImagePullSecrets) != 1 || sa.ImagePullSecrets[0].Name != "my-pull-secret" {
		t.Errorf("expected ImagePullSecrets [my-pull-secret], got %v", sa.ImagePullSecrets)
	}
	if sa.Annotations["iam.amazonaws.com/role"] != "arn:aws:iam::123:role/my-role" {
		t.Errorf("expected ClusterGuardController.ServiceAccount annotations to be propagated, got %v", sa.Annotations)
	}
}

// ClusterRoleBinding / RoleBinding builder tests

// assets.ClusterRoleBinding and assets.RoleBinding are covered in the assets package;
// these only check the role, ServiceAccount and namespace the controller passes in.
func TestAdmissionClusterRoleBindingWiring(t *testing.T) {
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "test-ns",
		},
	})
	crb := a.clusterRoleBinding()

	if crb.RoleRef.Name != common.ClusterGuardControllerClusterRoleName {
		t.Errorf("expected RoleRef.Name=%q, got %q", common.ClusterGuardControllerClusterRoleName, crb.RoleRef.Name)
	}
	if len(crb.Subjects) != 1 {
		t.Fatalf("expected 1 subject, got %d", len(crb.Subjects))
	}
	if crb.Subjects[0].Name != common.ClusterGuardControllerServiceAccountName {
		t.Errorf("expected subject Name=%q, got %q", common.ClusterGuardControllerServiceAccountName, crb.Subjects[0].Name)
	}
	if crb.Subjects[0].Namespace != "test-ns" {
		t.Errorf("expected subject Namespace=%q, got %q", "test-ns", crb.Subjects[0].Namespace)
	}
}

func TestAdmissionRoleBindingWiring(t *testing.T) {
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "test-ns",
		},
	})
	rb := a.roleBinding()

	if rb.RoleRef.Name != common.ClusterGuardControllerNamespaceRoleName {
		t.Errorf("expected RoleRef.Name=%q, got %q", common.ClusterGuardControllerNamespaceRoleName, rb.RoleRef.Name)
	}
	if len(rb.Subjects) != 1 {
		t.Fatalf("expected 1 subject, got %d", len(rb.Subjects))
	}
	if rb.Subjects[0].Name != common.ClusterGuardControllerServiceAccountName {
		t.Errorf("expected subject Name=%q, got %q", common.ClusterGuardControllerServiceAccountName, rb.Subjects[0].Name)
	}
	if rb.Subjects[0].Namespace != "test-ns" {
		t.Errorf("expected subject Namespace=%q, got %q", "test-ns", rb.Subjects[0].Namespace)
	}
}

// WebhookService builder tests

func TestAdmissionWebhookServiceSelectorUsesClusterGuardApp(t *testing.T) {
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "test-ns",
		},
	})
	svc := a.webhookService()

	if svc.Spec.Selector["app"] != common.ClusterGuardServiceApp {
		t.Errorf("expected selector app=%q, got %q", common.ClusterGuardServiceApp, svc.Spec.Selector["app"])
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

func TestAdmissionAPIServiceSelectorUsesClusterGuardApp(t *testing.T) {
	a := New(nil, Config{
		BaseConfig: components.BaseConfig{
			InstallNamespace: "test-ns",
		},
	})
	svc := a.apiService()

	if svc.Spec.Selector["app"] != common.ClusterGuardServiceApp {
		t.Errorf("expected selector app=%q, got %q", common.ClusterGuardServiceApp, svc.Spec.Selector["app"])
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
