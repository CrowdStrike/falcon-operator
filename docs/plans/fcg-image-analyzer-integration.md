# Plan: FalconImageAnalyzer Integration into FalconClusterGuard

## Execution Order

```
Step 1 (types + conditions + constants + static RBAC yaml)
    └─> Step 2 (component — 6 new files)
    └─> Step 3 (reconciler wiring)
    └─> Step 4 (FalconImageAnalyzer deprecation webhook)
            └─> Step 5 (make generate manifests)
```

Steps 2, 3, and 4 can be developed in parallel once Step 1 compiles cleanly.

---

## Step 1 — Extend Type Definitions

### `api/falcon/v1alpha1/falconclusterguard_types.go`

Add a new struct `FalconClusterGuardImageAnalyzerSpec`. Fields from `FalconImageAnalyzerConfigSpec` are inlined directly — no nested sub-struct. Fields excluded from this struct (inherited from parent `FalconClusterGuardSpec`): `ImagePullPolicy`, `ImagePullSecrets`. No per-component `Image` (all FCG components share `Spec.Image`) or `InstallNamespace` (all components share `Spec.InstallNamespace`).

```go
// FalconClusterGuardImageAnalyzerSpec configures the Image Analyzer Deployment managed by FalconClusterGuard.
type FalconClusterGuardImageAnalyzerSpec struct {
    // +kubebuilder:default:=false
    Enabled           *bool                                `json:"enabled,omitempty"`
    NodeAffinity      *corev1.NodeAffinity                 `json:"nodeAffinity,omitempty"`
    Tolerations       []corev1.Toleration                  `json:"tolerations,omitempty"`
    ServiceAccount    FalconImageAnalyzerServiceAccount    `json:"serviceAccount,omitempty"`
    Resources         *corev1.ResourceRequirements         `json:"resources,omitempty"`
    AzureConfigPath   string                               `json:"azureConfigPath,omitempty"`
    PriorityClass     FalconImageAnalyzerPriorityClass     `json:"priorityClass,omitempty"`
    // +kubebuilder:default:={}
    DepUpdateStrategy FalconImageAnalyzerUpdateStrategy    `json:"updateStrategy,omitempty"`
    // +kubebuilder:default:="20Gi"
    VolumeSizeLimit   string                               `json:"sizeLimit,omitempty"`
    // +kubebuilder:default:="/tmp"
    VolumeMountPath   string                               `json:"mountPath,omitempty"`
    ClusterName       string                               `json:"clusterName,omitempty"`
    Exclusions        Exclusions                           `json:"exclusions,omitempty"`
    // +kubebuilder:default:={}
    RegistryConfig    RegistryConfig                       `json:"registryConfig,omitempty"`
    // +kubebuilder:default:=false
    EnableDebug       bool                                 `json:"debug,omitempty"`
    // +kubebuilder:default:=info
    // +kubebuilder:validation:Enum=info;debug;warn;error
    LogVerbosity      string                               `json:"logVerbosity,omitempty"`
    // +kubebuilder:default:={}
    IARAgentService   FalconImageAnalyzerAgentServiceSpec  `json:"iarAgentService,omitempty"`
}

func (s *FalconClusterGuardImageAnalyzerSpec) IsEnabled() bool {
    return s.Enabled != nil && *s.Enabled
}
```

Fields excluded (inherited from parent `FalconClusterGuardSpec`): `ImagePullPolicy`, `ImagePullSecrets`.
`KAC` (`FalconImageAnalyzerKACSpec`) is also excluded — it only held a `Namespace` field. Since KAC (the cluster guard controller) always runs in the same namespace as IAR, the component uses `ia.cfg.InstallNamespace` directly for inter-communication configuration.

Add the field to `FalconClusterGuardSpec`:

```go
// +kubebuilder:default={}
// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Image Analyzer Configuration",order=10
ImageAnalyzer FalconClusterGuardImageAnalyzerSpec `json:"imageAnalyzer,omitempty"`
```

### `api/falcon/v1alpha1/conditions.go`

Add one constant:

```go
ConditionImageAnalyzerReady string = "ImageAnalyzerReady"
```

### `pkg/common/constants.go`

Add FCG-specific IAR constants (avoids name collision with standalone `FalconImageAnalyzer`):

```go
// FCG-owned Image Analyzer module constants
FCGImageAnalyzerServiceAccountName = "falcon-fcg-iar-sa"
FCGImageAnalyzerConfigMapName       = "falcon-fcg-iar-config"
FCGImageAnalyzerDeploymentName      = "falcon-fcg-image-analyzer"
FCGImageAnalyzerTLSSecretName       = "falcon-fcg-iar-tls"
FCGImageAnalyzerCRBName             = "falcon-fcg-iar-crb"
// "falcon-operator-" namePrefix from config/default/kustomization.yaml is applied at deploy time
FCGImageAnalyzerClusterRoleName     = "falcon-operator-falcon-image-analyzer-role"
FCGImageAnalyzerComponentName       = "fcg-iar"
```

### `config/rbac/falconclusterguard_image_analyzer_clusterrole.yaml` (new file)

The IAR pod's ServiceAccount is bound to a static ClusterRole deployed via kustomize — matching the existing pattern used by `node_sensor` (`falconclusterguard_sensor_clusterrole.yaml`) and `clusterguard_controller` (`falconclusterguard_admission_clusterrole.yaml`). The component reconciles only a **ClusterRoleBinding**; it never creates or modifies ClusterRoles at runtime.

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  labels:
    crowdstrike.com/component: rbac
    crowdstrike.com/created-by: falcon-operator
    crowdstrike.com/managed-by: kustomize
    crowdstrike.com/name: clusterrole
    crowdstrike.com/part-of: Falcon
    crowdstrike.com/provider: crowdstrike
  name: falcon-image-analyzer-role
rules:
- apiGroups: [""]
  resources: [namespaces, pods, secrets, services, nodes]
  verbs: [get, list, watch]
- apiGroups: [security.openshift.io]
  resourceNames: [privileged]
  resources: [securitycontextconstraints]
  verbs: [use]
```

Add to `config/rbac/kustomization.yaml` under the `# ClusterGuard RBAC` section:

```yaml
- falconclusterguard_image_analyzer_clusterrole.yaml
```

### RBAC impact on the FCG operator itself

| Resource | Current FCG markers | After this change |
|---|---|---|
| `pods` | not present | `get;list;watch;update` (new marker needed) |
| `clusterroles` | `get;list;watch` | unchanged — sufficient since no dynamic ClusterRole creation |
| `clusterrolebindings` | `create;get;list;update;watch;delete` | unchanged |
| everything else (namespaces, configmaps, secrets, serviceaccounts, services, deployments) | already present | unchanged |

---

## Step 2 — Create the `image_analyzer` Component

**Directory:** `internal/controller/components/image_analyzer/`
(stub directory already exists but is empty)

All six files use `package image_analyzer`.

### File Overview

| File | Purpose |
|---|---|
| `reconcile.go` | `Config` struct, `ImageAnalyzer` type, `New()`, `Reconcile()` |
| `rbac.go` | `reconcileServiceAccount`, `reconcileClusterRoleBinding` |
| `configmap.go` | Port configmap builder from `falcon_image_analyzer/configmap.go`; returns `(bool, error)` |
| `deployment.go` | Build deployment from flat fields (see note on assets coupling below) |
| `secrets.go` | Port `reconcileIARTLSSecret` logic |
| `services.go` | Port `reconcileIARAgentService` logic |

All logic from `falcon_image_analyzer/` is **copied** into the new package — those files are not modified. The existing functions are methods on `FalconImageAnalyzerReconciler` and cannot be called from the new component.

### `reconcile.go` — Key Structures

```go
type Config struct {
    components.BaseConfig
    ImageAnalyzerSpec falconv1alpha1.FalconClusterGuardImageAnalyzerSpec
    FalconAPI         *falconv1alpha1.FalconAPI
}

type ImageAnalyzer struct {
    r   k8sutils.Reconciler
    cfg Config
}

func New(r k8sutils.Reconciler, cfg Config) *ImageAnalyzer
func (ia *ImageAnalyzer) Reconcile(ctx context.Context) (ctrl.Result, error)
```

### `Reconcile()` Call Order

All resources are created in `ia.cfg.InstallNamespace` (the FCG install namespace, from `BaseConfig`).

1. `reconcileServiceAccount`
2. `reconcileClusterRoleBinding`
3. `reconcileConfigMap` → capture `configUpdated bool`
4. `reconcileTLSSecret`
5. `reconcileService`
6. `reconcileDeployment`
7. If `configUpdated`, bump deployment annotation to trigger rolling restart
8. Set `ConditionImageAnalyzerReady=True` on status

### CID Note

Pass `ia.cfg.Cid` (already in `BaseConfig`) to the configmap builder. The FCG reconciler resolves CID before calling any component, so no redundant Falcon API call is needed.

### `deployment.go` Note

Do **not** call `assets.ImageAnalyzerDeployment()` directly — it takes `*FalconImageAnalyzer`. Instead, add a new function alongside it:

```go
// in internal/controller/assets/deployment.go
func ImageAnalyzerDeploymentFromConfig(...flat fields...) *appsv1.Deployment
```

Fields are accessed directly from `ia.cfg.ImageAnalyzerSpec` (e.g. `ia.cfg.ImageAnalyzerSpec.Resources`, `ia.cfg.ImageAnalyzerSpec.DepUpdateStrategy`) — no nested sub-struct access. Leave the existing `ImageAnalyzerDeployment()` untouched for the standalone FIA controller.

---

## Step 3 — Wire into the FCG Reconciler

**`internal/controller/falcon_clusterguard/falconclusterguard_controller.go`**

### 3a — Add Import

```go
"github.com/crowdstrike/falcon-operator/internal/controller/components/image_analyzer"
```

### 3b — Add Component Call

After the `node_sensor` block, before the deletion-timestamp guard. The IAR component uses `BaseConfig.Image` — the same already-resolved image URI shared by all FCG components.

```go
if falconClusterGuard.Spec.ImageAnalyzer.IsEnabled() {
    if result, err := image_analyzer.New(r, image_analyzer.Config{
        BaseConfig:        base,
        ImageAnalyzerSpec: falconClusterGuard.Spec.ImageAnalyzer,
        FalconAPI:         falconClusterGuard.Spec.FalconAPI,
    }).Reconcile(ctx); err != nil || result.RequeueAfter > 0 {
        return result, err
    }
}
```

### 3c — Add RBAC Marker

One new marker needed on the FCG controller — `pods` is not currently present:

```go
//+kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;update
```

The existing `clusterroles: get;list;watch` marker is sufficient — the component only creates a ClusterRoleBinding referencing the static ClusterRole; it never creates or modifies ClusterRoles at runtime.

### 3d — Auto-populate `FalconImageAnalyzerNamespace`

Before building `ClusterGuardControllerConfig`, if IAR is enabled, set:

```go
falconClusterGuard.Spec.ClusterGuardControllerConfig.FalconImageAnalyzerNamespace =
    falconClusterGuard.Spec.InstallNamespace
```

This ensures KAC knows where to find the FCG-deployed IAR without the user having to set it separately.

---

## Step 4 — Deprecate `FalconImageAnalyzer` via Webhook

`FalconImageAnalyzer` has no deprecation webhook today. `FalconNodeSensor` and `FalconAdmission` already follow this pattern; `FalconImageAnalyzer` should match.

### `api/falcon/v1alpha1/falconimageanalyzer_webhook.go` (new file)

Blocks `create` and `update` with `failurePolicy=ignore`, matching the `FalconAdmission` pattern:

```go
package v1alpha1

import (
    "context"
    "fmt"

    ctrl "sigs.k8s.io/controller-runtime"
    "sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

//+kubebuilder:webhook:path=/validate-falcon-crowdstrike-com-v1alpha1-falconimageanalyzer,mutating=false,failurePolicy=ignore,sideEffects=None,groups=falcon.crowdstrike.com,resources=falconimageanalyzers,verbs=create;update,versions=v1alpha1,name=vfalconimageanalyzer.kb.io,admissionReviewVersions=v1

// FalconImageAnalyzerValidator blocks creation of new FalconImageAnalyzer objects.
// FalconImageAnalyzer is deprecated; use FalconClusterGuard instead.
type FalconImageAnalyzerValidator struct{}

var _ admission.Validator[*FalconImageAnalyzer] = &FalconImageAnalyzerValidator{}

func (v *FalconImageAnalyzerValidator) SetupWebhookWithManager(mgr ctrl.Manager) error {
    return ctrl.NewWebhookManagedBy(mgr, &FalconImageAnalyzer{}).
        WithValidator(v).
        Complete()
}

func (v *FalconImageAnalyzerValidator) ValidateCreate(_ context.Context, _ *FalconImageAnalyzer) (admission.Warnings, error) {
    return nil, fmt.Errorf("FalconImageAnalyzer is deprecated; use FalconClusterGuard instead")
}

func (v *FalconImageAnalyzerValidator) ValidateUpdate(_ context.Context, _, _ *FalconImageAnalyzer) (admission.Warnings, error) {
    return nil, fmt.Errorf("FalconImageAnalyzer is deprecated; use FalconClusterGuard instead")
}

func (v *FalconImageAnalyzerValidator) ValidateDelete(_ context.Context, _ *FalconImageAnalyzer) (admission.Warnings, error) {
    return nil, nil
}
```

### `cmd/main.go`

Register the new validator alongside the existing ones (inside the `if cfg.EnableWebhooks` block):

```go
if err := (&falconv1alpha1.FalconImageAnalyzerValidator{}).SetupWebhookWithManager(mgr); err != nil {
    setupLog.Error(err, "unable to create webhook", "webhook", "FalconImageAnalyzer")
    os.Exit(1)
}
```

### `config/webhook/manifests.yaml`

`make generate manifests` will auto-add the entry from the `//+kubebuilder:webhook` marker. No manual edit needed.

---

## Step 5 — CRD and Manifest Regeneration

```bash
make generate manifests
```

Regenerates:
- `config/crd/bases/falcon.crowdstrike.com_falconclusterguards.yaml`
- `bundle/manifests/falcon.crowdstrike.com_falconclusterguards.yaml`
- `config/webhook/manifests.yaml` — picks up the new `FalconImageAnalyzer` webhook entry
- RBAC manifests from the new `//+kubebuilder:rbac` markers

---

## Anticipated Challenges

### 1. `assets.ImageAnalyzerDeployment` coupling
The existing function takes `*FalconImageAnalyzer`. Add `ImageAnalyzerDeploymentFromConfig(...)` accepting flat fields alongside it; leave the existing function intact for the standalone FIA controller.
