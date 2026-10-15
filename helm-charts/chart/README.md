# falcon-operator Helm Chart

Deploys the [CrowdStrike Falcon Operator](https://github.com/crowdstrike/falcon-operator) onto a Kubernetes cluster.

## Prerequisites

- Kubernetes 1.25+
- Helm 3.x

## Installation

```bash
helm install falcon-operator ./helm-charts/chart \
  --namespace falcon-operator \
  --create-namespace
```

To override the operator image tag:

```bash
helm install falcon-operator ./helm-charts/chart \
  --namespace falcon-operator \
  --create-namespace \
  --set manager.image.tag=<tag>
```

## Upgrading

For routine upgrades, run `helm upgrade --install` against the existing release. Because CRDs are managed as regular Helm-tracked resources (under `templates/crd/`), they are updated in-place alongside RBAC, the manager Deployment, and all other chart resources in a single operation:

```bash
helm upgrade --install falcon-operator ./helm-charts/chart \
  --namespace falcon-operator
```

For major version upgrades that may include breaking CRD schema changes, back up existing custom resources first:

```bash
kubectl get falcondeployment,falconnodesensor,falconadmission,falconclusterguard \
  -A -o yaml > falcon-cr-backup.yaml
```

Then run the upgrade. If any existing CRs are invalid against the new schema, the backup lets you restore or migrate them manually.

> **Note:** `helm upgrade --install` also handles the case where the release does not yet exist, making it safe to use in CI pipelines for both initial installs and upgrades.

## Uninstallation

```bash
helm uninstall falcon-operator --namespace falcon-operator
```

> **Note:** By default (`crd.keep=false`) CRDs are deleted on uninstall, which cascades and removes all existing CR instances (FalconNodeSensor, FalconDeployment, etc.). Set `crd.keep=true` to retain CRDs and preserve those resources across reinstalls or upgrades. **However**, since CRDs are tracked as regular Helm resources, a subsequent `helm install` will fail with a conflict because the CRDs already exist but are no longer owned by any release. To reinstall after keeping CRDs, apply any CRD schema updates manually with `kubectl apply -f` before running `helm install`.

## Values

### Manager

| Key | Default | Description |
|-----|---------|-------------|
| `manager.enabled` | `true` | Deploy the controller manager |
| `manager.replicas` | `1` | Number of replicas |
| `manager.image.repository` | `quay.io/crowdstrike/falcon-operator` | Image repository |
| `manager.image.tag` | `""` | Image tag (defaults to `Chart.appVersion`); set automatically by `make helm-build` |
| `manager.image.pullPolicy` | `Never` | Image pull policy |
| `manager.args` | `[--leader-elect]` | Extra arguments passed to the manager |
| `manager.env` | see values.yaml | Environment variables |
| `manager.envOverrides` | `{}` | Per-variable env overrides (`--set manager.envOverrides.FOO=bar`) |
| `manager.resources.limits.cpu` | `500m` | CPU limit |
| `manager.resources.limits.memory` | `256Mi` | Memory limit |
| `manager.resources.requests.cpu` | `100m` | CPU request |
| `manager.resources.requests.memory` | `64Mi` | Memory request |
| `manager.podSecurityContext` | see values.yaml | Pod-level security context |
| `manager.securityContext` | see values.yaml | Container-level security context |
| `manager.affinity` | linux amd64/arm64/ppc64le/s390x | Node affinity rules |
| `manager.nodeSelector` | `{}` | Node selector |
| `manager.tolerations` | `[]` | Tolerations |
| `manager.terminationGracePeriodSeconds` | `10` | Termination grace period |
| `manager.extraVolumeMounts` | `[]` | Extra volume mounts appended to the manager container |
| `manager.extraVolumes` | `[]` | Extra volumes appended to the manager pod |

### ServiceAccount

| Key | Default | Description |
|-----|---------|-------------|
| `serviceAccount.enabled` | `true` | Create a ServiceAccount for the manager |

### CRDs

| Key | Default | Description |
|-----|---------|-------------|
| `crd.enabled` | `true` | Install CRDs with the chart |
| `crd.keep` | `false` | Retain CRDs when the release is uninstalled |

### Webhook

| Key | Default | Description |
|-----|---------|-------------|
| `webhook.enabled` | `true` | Deploy the validating webhook server. When set to `false`, also add `--enable-webhooks=false` to `manager.args`. Without that flag the manager still starts its webhook HTTP server on port 9443 — leaving an open port with no functional purpose and unnecessary attack surface inside the cluster. |

## Manager Args (`manager.args`)

Flags passed directly to the manager binary. The default list is `[--leader-elect]`.

| Flag | Default | Description |
|------|---------|-------------|
| `--leader-elect` | `false` | Enable leader election for high availability |
| `--enable-webhooks` | `true` | Enable CRD validation webhooks. Set to `false` to skip cert generation and webhook registration — required when `webhook.enabled=false` |
| `--metrics-bind-address` | `0` | Address the metrics endpoint binds to (`0` disables it) |
| `--metrics-secure` | `true` | Serve metrics over HTTPS with authentication |
| `--health-probe-bind-address` | `:8081` | Address the liveness/readiness probe endpoint binds to |
| `--profile` | `false` | Enable pprof profiling |
| `--profile-bind-address` | `localhost:8082` | Address the profiling endpoint binds to |
| `--enable-http2` | `false` | Enable HTTP/2 on the webhook and metrics servers (disabled by default to mitigate HTTP/2 stream-reset vulnerabilities) |
| `--openshift` | `false` | Force OpenShift mode. If unset, OpenShift is auto-detected |
| `--sensor-auto-update-interval` | `24h` | How often the Falcon API is queried for new sensor versions |
| `--lease-duration` | `30s` | How long non-leader candidates wait before forcing leadership acquisition |
| `--renew-deadline` | `20s` | How long the acting leader retries refreshing leadership before giving up |
| `--webhook-service-name` | `falcon-operator-webhook-service` | Name of the Service fronting the webhook server |
| `--webhook-config-name` | `falcon-operator-validating-webhook-configuration` | Name of the `ValidatingWebhookConfiguration` managed by the operator |
| `--webhook-cert-secret` | `webhook-server-cert` | Name of the Secret containing the webhook TLS certificate. Create this Secret manually to provide a custom certificate, or point it at a cert-manager-managed Secret. Note: the cert is read once at startup — a manager restart is required to pick up cert-manager renewals |
| `--version` | `false` | Print version and exit |
