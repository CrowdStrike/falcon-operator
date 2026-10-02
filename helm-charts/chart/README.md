# falcon-operator Helm Chart

Deploys the [CrowdStrike Falcon Operator](https://github.com/crowdstrike/falcon-operator) onto a Kubernetes cluster.

## Prerequisites

- Kubernetes 1.25+
- Helm 3.x
- [cert-manager](https://cert-manager.io/) installed in the cluster (required for webhook TLS certificates)

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

## Uninstallation

```bash
helm uninstall falcon-operator --namespace falcon-operator
```

> **Note:** CRDs are kept on uninstall by default when `crd.keep=true`. Set `crd.keep=false` (the default) to remove them.

## Values

### Manager

| Key | Default | Description |
|-----|---------|-------------|
| `manager.enabled` | `true` | Deploy the controller manager |
| `manager.replicas` | `1` | Number of replicas |
| `manager.image.repository` | `quay.io/crowdstrike/falcon-operator` | Image repository |
| `manager.image.tag` | `""` | Image tag (defaults to `Chart.appVersion`) |
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

### RBAC

| Key | Default | Description |
|-----|---------|-------------|
| `rbac.namespaced` | `false` | `false` = ClusterRole/ClusterRoleBinding; `true` = Role/RoleBinding scoped to the release namespace |
| `rbac.helpers.enabled` | `false` | Install admin/editor/viewer helper roles for CRDs |

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
| `webhook.enabled` | `true` | Deploy the webhook server |
| `webhook.port` | `443` | Webhook server port |

### cert-manager

| Key | Default | Description |
|-----|---------|-------------|
| `certManager.enabled` | `true` | Use cert-manager to manage TLS certificates for the webhook |

### Metrics

| Key | Default | Description |
|-----|---------|-------------|
| `metrics.enabled` | `false` | Expose the `/metrics` endpoint |
| `metrics.port` | `8443` | Metrics server port |
| `metrics.secure` | `true` | Serve metrics over HTTPS with authentication |

### Prometheus

| Key | Default | Description |
|-----|---------|-------------|
| `prometheus.enabled` | `false` | Create a Prometheus `ServiceMonitor` (requires prometheus-operator) |

### Network Policy

| Key | Default | Description |
|-----|---------|-------------|
| `networkPolicy.enabled` | `false` | Create a `NetworkPolicy` restricting ingress to the manager |
