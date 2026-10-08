# Falcon Cluster Guard

## About FalconClusterGuard Custom Resource (CR)
Falcon Operator introduces the FalconClusterGuard Custom Resource (CR) to the cluster. The resource is meant to install, configure, and uninstall Falcon Cluster Guard on the cluster. FalconClusterGuard combines the capabilities of the Falcon Kubernetes Admission Controller, the Falcon Linux Node Sensor, and the Falcon Image Analyzer into a single resource, deploying a Deployment (for admission control and cluster visibility), a DaemonSet (for node-level protection), and optionally an Image Analyzer Deployment (for image assessment at runtime) from a single manifest.

> [!IMPORTANT]
> `FalconNodeSensor`, `FalconAdmission`, and `FalconImageAnalyzer` are deprecated. New deployments should use `FalconClusterGuard` instead. Existing `FalconNodeSensor`, `FalconAdmission`, and `FalconImageAnalyzer` resources can be deleted and replaced with a `FalconClusterGuard` resource. Creation of new `FalconNodeSensor` resources and creation or update of `FalconAdmission` and `FalconImageAnalyzer` resources are blocked by a validating webhook.

### FalconClusterGuard CR Configuration using CrowdStrike API Keys
To start the FalconClusterGuard installation using CrowdStrike API Keys to allow the operator to determine your Falcon Customer ID (CID) as well as pull down the CrowdStrike Falcon Cluster Guard image, please create the following FalconClusterGuard resource to your cluster.

> [!IMPORTANT]
> You will need to provide CrowdStrike API Keys and CrowdStrike cloud region for the installation. It is recommended to establish new API credentials for the installation at https://falcon.crowdstrike.com/support/api-clients-and-keys, required permissions are:
> * Falcon Images Download: **Read**
> * Sensor Download: **Read**
>
> When the Image Analyzer is enabled (`imageAnalyzer.enabled: true`), the following permissions are also required:
> * Falcon Container CLI: **Write**
> * Falcon Container Image: **Read/Write**

Example:

```yaml
apiVersion: falcon.crowdstrike.com/v1alpha1
kind: FalconClusterGuard
metadata:
  name: falcon-clusterguard
spec:
  falcon:
    tags:
      - test-cluster
      - dev
  falcon_api:
    client_id: PLEASE_FILL_IN
    client_secret: PLEASE_FILL_IN
    cloud_region: autodiscover
  registry:
    type: crowdstrike
```

To also deploy the Falcon Image Analyzer, enable it under `imageAnalyzer`:

```yaml
apiVersion: falcon.crowdstrike.com/v1alpha1
kind: FalconClusterGuard
metadata:
  name: falcon-clusterguard
spec:
  falcon_api:
    client_id: PLEASE_FILL_IN
    client_secret: PLEASE_FILL_IN
    cloud_region: autodiscover
  registry:
    type: crowdstrike
  imageAnalyzer:
    enabled: true
    clusterName: PLEASE_FILL_IN
```

### FalconClusterGuard Reference Manual

#### Falcon API Settings
| Spec                     | Description                                                                                                                                                                                                                          |
|:-------------------------|:-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| falcon_api.client_id     | (optional) CrowdStrike API Client ID                                                                                                                                                                                                 |
| falcon_api.client_secret | (optional) CrowdStrike API Client Secret                                                                                                                                                                                             |
| falcon_api.cloud_region  | (optional) CrowdStrike cloud region (allowed values: autodiscover, us-1, us-2, eu-1, us-gov-1, us-gov-2);<br> Falcon API credentials or [Falcon Secret with credentials](#falcon-secret-settings) are required if `cloud_region: autodiscover`;<br> `autodiscover` cannot be used for us-gov-1 or us-gov-2 |
| falcon_api.cid           | (optional) CrowdStrike Falcon CID API override; Required for us-gov-2                                                                                                                                                                |

#### FalconClusterGuard Configuration Settings
| Spec                      | Description                                                                                                                                                                                     |
|:--------------------------|:------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| installNamespace          | (optional) Override the default namespace of `falcon-system`                                                                                                                                    |
| image                     | (optional) Leverage a Falcon Cluster Guard image that is not managed by the operator; typically used with custom repositories; overrides all registry settings; might require imagePullSecrets to be set |
| version                   | (optional) Enforce particular Falcon Cluster Guard version to be installed (example: "7.33", "7.33.0", "7.33.0-1409")                                                                          |
| imagePullPolicy           | (optional) Override the default image pull policy of IfNotPresent                                                                                                                               |
| imagePullSecrets          | (optional) List of references to secrets to use for pulling the image                                                                                                                           |
| registry.type                       | Specifies the source registry for the Falcon Cluster Guard image. Use `crowdstrike` to have the operator create an image pull secret for the CrowdStrike registry (default). Use `private` when the image is hosted in a private registry and image pull credentials are managed separately via `imagePullSecrets`. |
| registry.tls.insecure_skip_verify   | (optional) Skip TLS verification when connecting to the container image registry                                                                                                              |
| registry.tls.caCertificate          | (optional) A string containing an optionally base64-encoded Certificate Authority Chain for connecting to a registry with a self-signed TLS certificate                                       |
| registry.tls.caCertificateConfigMap | (optional) The name of a ConfigMap containing CA Certificate Authority Chains under keys ending in ".tls" for connecting to a registry with a self-signed TLS certificate (ignored when registry.tls.caCertificate is set) |

#### Cluster Guard Controller Configuration Settings
| Spec                                              | Description                                                                                                                                                                                                  |
|:--------------------------------------------------|:-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| controller.disableClusterGuardController   | (optional) To disable the Falcon Cluster Guard Controller, set this field to the acknowledgment string: `"I understand this disables K8s metadata collection and degrades node sensor visibility"`. Omit this field (default) to keep the controller enabled. Disabling the controller on an existing resource removes its resources from the cluster. |
| controller.clusterName                     | (optional) Custom cluster name to be used by the controller if automatic discovery fails. Note that this value cannot be changed after initial deployment and requires a full redeployment to modify. |
| controller.serviceAccount.annotations     | (optional) Configure annotations for the controller service account (e.g. for IAM role association)                                                                                                |
| controller.servicePort                     | (optional) Configure the port the controller Service listens on. Default is `443`.                                                                                                                 |
| controller.containerPort                   | (optional) Configure the port the controller container listens on. Default is `4443`.                                                                                                              |
| controller.watcherPort                     | (optional) Configure the port the falcon-watcher container listens on. Default is `4080`.                                                                                                          |
| controller.tls.validity                    | (optional) Configure the validity of the TLS certificate used by the controller.                                                                                                                   |
| controller.tlsVersionMinimum               | (optional) Minimum TLS version accepted by the webhook server. Valid values: `TLS1.2`, `TLS1.3`. When unset the server default is used.                                                            |
| controller.resourceQuotaPods               | (optional) Maximum number of pods allowed with the `system-cluster-critical` priority class. Controls the ResourceQuota for the admission controller namespace. Default is `2`.                    |
| controller.failurePolicy                   | (optional) Configure the failure policy of the admission webhook (`Ignore` or `Fail`). Default is `Ignore`.                                                                                        |
| controller.disabledNamespaces.namespaces   | (optional) Configure the list of namespaces the admission webhook should ignore.                                                                                                                   |
| controller.watcherEnabled                  | (optional) Determines if Kubernetes resources are watched for cluster visibility. Default is `true`.                                                                                               |
| controller.snapshotsEnabled                | (optional) Determines if snapshots of Kubernetes resources are periodically taken for cluster visibility. Default is `true`.                                                                       |
| controller.snapshotsInterval               | (optional) Time interval between two snapshots of Kubernetes resources in the cluster. Default is `22h`.                                                                                           |
| controller.configMapWatcherEnabled         | (optional) Determines if the watcher for ConfigMap events is enabled. Default is `true`.                                                                                                           |
| controller.admissionControlEnabled         | (optional) Enable the admission webhook. Default is `false`.                                                                                                                                       |
| controller.replicas                        | (optional) Ignored — the controller always runs as a single replica.                                                                                                                               |
| controller.resourcesClient                 | (optional) Configure the resources for the falcon-client container.                                                                                                                                |
| controller.resourcesClientNoWebhook        | (optional) Configure the resources for the falcon-client container only when the admission webhook is disabled. Overrides `controller.resourcesClient`.                                      |
| controller.resourcesWatcher                | (optional) Configure the resources for the falcon-watcher container.                                                                                                                               |
| controller.resources                       | (optional) Configure the resources for the falcon-ac container.                                                                                                                                    |
| controller.updateStrategy                  | (optional) Configure the Deployment update strategy for the controller.                                                                                                                            |
| controller.nodeAffinity                    | (optional) See https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/ for examples on configuring nodeAffinity. AMD64 and ARM64 architectures are supported by default.          |
| controller.tolerations                     | (optional) Specify tolerations for scheduling the controller pods. See https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/ for examples. Note: Tolerations can be added or updated through the operator, but removing tolerations from the spec requires manual deletion from the deployment to distinguish user-defined tolerations from those automatically added by Kubernetes. |
| controller.falconImageAnalyzerNamespace    | (optional) Namespace where a standalone Falcon Image Analyzer is installed. Required only if it is not `falcon-iar`. Ignored when `imageAnalyzer.enabled` is true (IAR runs in `installNamespace`). |

> [!IMPORTANT]
> Always install Falcon Cluster Guard to its own unique namespace. We recommend the namespace `falcon-system`. The controller does not monitor its own namespace.

#### Node Sensor Configuration Settings
| Spec                                        | Description                                                                                                                                                                                      |
|:--------------------------------------------|:-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| nodeSensor.enabled                          | (optional) Controls whether the node sensor DaemonSet is deployed. Default is `true`. Setting it to `false` on an existing resource runs node cleanup (unless `disableCleanup` is set) and removes the node sensor resources. |
| nodeSensor.tolerations                      | (optional) See https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/ for examples on configuring tolerations. Note: Tolerations can be added or updated through the operator, but removing tolerations from the spec requires manual deletion from the daemonset to distinguish user-defined tolerations from those automatically added by Kubernetes. |
| nodeSensor.nodeAffinity                     | (optional) See https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/ for examples on configuring nodeAffinity                                                                 |
| nodeSensor.updateStrategy                   | (optional) Configure the DaemonSet update strategy (RollingUpdate or OnDelete). Default is RollingUpdate.                                                                                        |
| nodeSensor.terminationGracePeriod           | (optional) Kills pod after a specified amount of time (in seconds). Default is 60 seconds.                                                                                                       |
| nodeSensor.serviceAccount.annotations      | (optional) Annotations that should be added to the node sensor Service Account (e.g. for IAM role association)                                                                                   |
| nodeSensor.disableCleanup                   | (optional) Disables the cleanup of `/opt/CrowdStrike` on nodes when the resource is deleted or the node sensor is disabled.                                                                      |
| nodeSensor.resources.limits.cpu             | (optional) CPU limit for the sensor DaemonSet. Minimum: `250m`. Only applies when using the eBPF backend.                                                                                        |
| nodeSensor.resources.limits.memory          | (optional) Memory limit for the sensor DaemonSet. Minimum: `500Mi`. Only applies when using the eBPF backend.                                                                                    |
| nodeSensor.resources.limits.ephemeral-storage | (optional) Ephemeral storage limit for the sensor DaemonSet. Only applies when using the eBPF backend.                                                                                         |
| nodeSensor.resources.requests.cpu           | (optional) CPU request for the sensor DaemonSet. Minimum: `250m`. Only applies when using the eBPF backend.                                                                                      |
| nodeSensor.resources.requests.memory        | (optional) Memory request for the sensor DaemonSet. Minimum: `500Mi`. Only applies when using the eBPF backend.                                                                                  |
| nodeSensor.resources.requests.ephemeral-storage | (optional) Ephemeral storage request for the sensor DaemonSet. Only applies when using the eBPF backend.                                                                                    |
| nodeSensor.backend                          | **(Deprecated)** This field is ignored. The sensor uses eBPF by default, but falls back to kernel mode on unsupported kernel versions.                                                           |
| nodeSensor.gke.autopilot                    | (optional) Enable GKE Autopilot support for the node sensor DaemonSet.                                                                                                                           |
| nodeSensor.gke.deployAllowListVersion       | (optional) WorkloadAllowlist version for the sensor daemonset when using GKE AutoPilot (example: "v1.0.3").                                                                                      |
| nodeSensor.gke.cleanupAllowListVersion      | (optional) WorkloadAllowlist version for the cleanup daemonset when using GKE AutoPilot (example: "v1.0.2").                                                                                     |
| nodeSensor.priorityClass.deploy             | (optional) Deploy a PriorityClass for the node sensor DaemonSet. Default is false.                                                                                                               |
| nodeSensor.priorityClass.name               | (optional) Name of the PriorityClass to use for the node sensor DaemonSet.                                                                                                                       |
| nodeSensor.priorityClass.value              | (optional) Value of the PriorityClass. Requires `priorityClass.deploy` to be `true`.                                                                                                             |
| nodeSensor.clusterName                      | (optional) When running on an unmanaged K8S cluster, set a cluster name. When running on managed K8S (e.g. EKS, GKE, AKS), cluster name is resolved cloud-side.                                 |
| nodeSensor.version                          | (optional) Reserved for future use. In FalconClusterGuard, sensor version is controlled by the top-level `version` field which applies to both components. This field has no effect.            |
| nodeSensor.advanced.autoUpdate              | (optional) Reserved for future use. Automatic sensor update tracking is not yet implemented for FalconClusterGuard. Use the top-level `version` field to pin a specific sensor version.         |
| nodeSensor.advanced.updatePolicy            | (optional) Reserved for future use. Sensor update policies are not yet implemented for FalconClusterGuard. Use the top-level `version` field to pin a specific sensor version.                  |

> [!IMPORTANT]
> nodeSensor.tolerations will be appended to the existing tolerations for the daemonset. Removing tolerations from an existing daemonset requires a redeploy of the FalconClusterGuard manifest.

#### Image Analyzer Configuration Settings
The Image Analyzer runs in `installNamespace` and uses the top-level `image`, `version`, `registry`, `imagePullPolicy`, and `imagePullSecrets` settings shared by all Falcon Cluster Guard components.

| Spec                                                  | Description                                                                                                                                                                                      |
|:------------------------------------------------------|:-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| imageAnalyzer.enabled                                 | (optional) Controls whether the Image Analyzer Deployment is deployed. Default is `false`. Setting it to `false` on an existing resource removes the Image Analyzer resources.                  |
| imageAnalyzer.clusterName                             | (optional) Name of the cluster as it will appear in the Falcon console.                                                                                                                         |
| imageAnalyzer.nodeAffinity                            | (optional) See https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/ for examples on configuring nodeAffinity. AMD64 and ARM64 architectures are supported by default.        |
| imageAnalyzer.tolerations                             | (optional) Specify tolerations for scheduling the Image Analyzer pods. See https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/ for examples. Note: Tolerations can be added or updated through the operator, but removing tolerations from the spec requires manual deletion from the deployment to distinguish user-defined tolerations from those automatically added by Kubernetes. |
| imageAnalyzer.serviceAccount.annotations             | (optional) Configure annotations for the Image Analyzer service account (e.g. for IAM role association). Note: Annotations can be added or updated through the operator, but removing existing annotations requires manual intervention. |
| imageAnalyzer.resources                               | (optional) Configure the resources of the Image Analyzer container.                                                                                                                             |
| imageAnalyzer.updateStrategy                          | (optional) Configure the Deployment update strategy of the Image Analyzer.                                                                                                                      |
| imageAnalyzer.priorityClass.name                      | (optional) Name of an existing PriorityClass to use for the Image Analyzer pods; set to avoid pod evictions due to resource limits.                                                             |
| imageAnalyzer.azureConfigPath                         | (optional) Path to the Azure config file on the node. Required only on AKS.                                                                                                                     |
| imageAnalyzer.sizeLimit                               | (optional) Configure the size limit of the temp storage space for scanning. Default is `20Gi`.                                                                                                  |
| imageAnalyzer.mountPath                               | (optional) Configure the location of the temp storage space for scanning. Default is `/tmp`.                                                                                                    |
| imageAnalyzer.logVerbosity                            | (optional) Set the log verbosity level. Allowed values: `info`, `debug`, `warn`, `error`. Default is `info`.                                                                                    |
| imageAnalyzer.debug                                   | (optional) **Deprecated** as of IAR 1.0.26. Use `imageAnalyzer.logVerbosity` instead. Set to `true` for debug level log.                                                                       |
| imageAnalyzer.exclusions.registries                   | (optional) List of registries to exclude. All images in the listed registries will be excluded.                                                                                                 |
| imageAnalyzer.exclusions.namespaces                   | (optional) List of namespaces to exclude. All pods in the listed namespaces will be excluded.                                                                                                   |
| imageAnalyzer.exclusions.imageNames                   | (optional) List of fully qualified image names to exclude.                                                                                                                                       |
| imageAnalyzer.registryConfig.credentials              | (optional) Registry secrets in the form of a list of maps. e.g.<pre>- namespace: ns1<br>&nbsp;&nbsp;secretName: mysecretname</pre>To scan OpenShift control plane components, specify the cluster's pull secret: <pre>- namespace: openshift-config<br>&nbsp;&nbsp;secretName: pull-secret</pre> |
| imageAnalyzer.registryConfig.autoDiscoverCredentials | (optional) Enable auto-discovery of registry credentials from secrets in the cluster. Default is `true`.                                                                                        |
| imageAnalyzer.iarAgentService.port                    | (optional) HTTPS port for the IAR Agent Service. Default is `8001`.                                                                                                                             |
| imageAnalyzer.iarAgentService.certExpiration          | (optional) Certificate validity duration in days. Default is `3650`.                                                                                                                            |
| imageAnalyzer.iarAgentService.domainName              | (optional) Custom DNS domain for services when .svc requires a domain (e.g., if service.my-namespace.svc doesn't resolve and the cluster uses service.my-namespace.svc.testing.io, add testing.io). |

> [!NOTE]
> When `imageAnalyzer.enabled` is `true`, the Cluster Guard Controller communicates with the Image Analyzer in `installNamespace` and `controller.falconImageAnalyzerNamespace` is ignored. Do not run a standalone `FalconImageAnalyzer` alongside the Image Analyzer managed by FalconClusterGuard.

#### Falcon Sensor Settings
| Spec                      | Description                                                                                                                                                                                                                  |
|:--------------------------|:-----------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| falcon.cid                | (optional) CrowdStrike Falcon CID override;<br> [Falcon API credentials](#falcon-api-settings) or [Falcon Secret with credentials](#falcon-secret-settings) are required if this field is not set;<br> Required for us-gov-2 |
| falcon.apd                | (optional) Disable the Falcon Sensor's use of a proxy.                                                                                                                                                                       |
| falcon.aph                | (optional) The application proxy host to use for Falcon sensor proxy configuration.                                                                                                                                          |
| falcon.app                | (optional) The application proxy port to use for Falcon sensor proxy configuration.                                                                                                                                          |
| falcon.billing            | (optional) Utilize default or Pay-As-You-Go billing.                                                                                                                                                                         |
| falcon.provisioning_token | (optional) Installation token that prevents unauthorized hosts from being accidentally or maliciously added to your customer ID (CID).                                                                                       |
| falcon.tags               | (optional) Sensor grouping tags are optional, user-defined identifiers that can be used to group and filter hosts. Allowed characters: all alphanumerics, '/', '-', and '_'.                                                  |
| falcon.trace              | (optional) Set sensor trace level (none, err, warn, info, debug).                                                                                                                                                            |

> [!IMPORTANT]
> All arguments are optional, but successful deployment requires either **client_id and client_secret or the Falcon cid and image**. When deploying using the CrowdStrike Falcon API, the container image and CID will be fetched from CrowdStrike Falcon API. While in the latter case, the CID and image location is explicitly specified by the user.

#### Falcon Secret Settings
| Spec                    | Description                                                                                    |
|:------------------------|:-----------------------------------------------------------------------------------------------|
| falconSecret.enabled    | Enable reading sensitive Falcon API and Falcon sensor values from k8s secret; Default: `false` |
| falconSecret.namespace  | Required if `enabled: true`; k8s namespace with relevant k8s secret                            |
| falconSecret.secretName | Required if `enabled: true`; name of k8s secret with sensitive Falcon API and sensor values    |

Falcon secret settings are used to read the following sensitive Falcon API and sensor values from an existing k8s secret on your cluster.

> [!IMPORTANT]
> When Falcon Secret is enabled, ALL spec parameters in the list of [secret keys](#secret-keys) will be overwritten.
> If a key/value does not exist in your k8s secret, the value will be overwritten as an empty string.

##### Secret Keys
| Secret Key                | Description                                                                                   |
|:--------------------------|:----------------------------------------------------------------------------------------------|
| falcon-client-id          | Replaces [`falcon_api.client_id`](#falcon-api-settings)                                       |
| falcon-client-secret      | Replaces [`falcon_api.client_secret`](#falcon-api-settings)                                   |
| falcon-cid                | Replaces [`falcon_api.cid`](#falcon-api-settings) and [`falcon.cid`](#falcon-sensor-settings) |
| falcon-provisioning-token | Replaces [`falcon.provisioning_token`](#falcon-sensor-settings)                               |

Example of creating k8s secret with sensitive Falcon values:
```bash
kubectl create secret generic falcon-secrets -n $FALCON_SECRET_NAMESPACE \
--from-literal=falcon-client-id=$FALCON_CLIENT_ID \
--from-literal=falcon-client-secret=$FALCON_CLIENT_SECRET \
--from-literal=falcon-cid=$FALCON_CID \
--from-literal=falcon-provisioning-token=$FALCON_PROVISIONING_TOKEN
```

### Auto Proxy Configuration

The operator will automatically configure the sensor's proxy configuration when the cluster proxy is configured on OpenShift via OLM. See the following documentation for more information:
* [Configuring cluster-wide proxy](https://docs.openshift.com/container-platform/latest/networking/enable-cluster-wide-proxy.html)
* [Overriding proxy settings of an Operator](https://docs.openshift.com/container-platform/4.13/operators/admin/olm-configuring-proxy-support.html#olm-overriding-proxy-settings_olm-configuring-proxy-support)

When not running on OpenShift, adding the proxy configuration via environment variables will also configure the sensor's proxy information.
```yaml
- args:
  - --leader-elect
  command:
  - /manager
  env:
  - name: OPERATOR_NAME
    value: falcon-operator
  - name: HTTP_PROXY
    value: http://proxy.example.com:8080
  - name: HTTPS_PROXY
    value: http://proxy.example.com:8080
  image: quay.io/crowdstrike/falcon-operator:latest
```
These settings can be overridden by configuring the [sensor's proxy settings](#falcon-sensor-settings) which will only change the sensor's proxy settings **not** the operator's proxy settings.

>[!IMPORTANT]
> 1. If using the CrowdStrike API with the **client_id and client_secret** authentication method, the operator must be able to reach the CrowdStrike API through the proxy via the Kubernetes cluster networking configuration.
>    If the proxy is not configured correctly, the operator will not be able to authenticate with the CrowdStrike API and will not be able to create the sensor.
> 2. If the CrowdStrike API is not used, configure the [sensor's proxy settings](#falcon-sensor-settings).
> 3. Ensure that the host node can reach the CrowdStrike Falcon Cloud through the proxy.


### Image Registry considerations

Falcon Cluster Guard image is pulled directly from the CrowdStrike Falcon registry or a private registry. The operator supports two modes:

#### (Option 1) Use CrowdStrike registry directly

The operator creates an image pull secret to authenticate with the CrowdStrike registry on your behalf. Use this when Falcon API credentials are provided.

```yaml
registry:
  type: crowdstrike
```

#### (Option 2) Use a private registry

Set `registry.type: private` and supply image pull credentials via `imagePullSecrets`. The operator will not create any pull secret. Use `image` to specify the full image URI.

```yaml
registry:
  type: private
image: myprivateregistry.internal.lan/falcon-clusterguard/falcon-sensor:7.33.0-1234.container.x86_64.Release.US-1
imagePullSecrets:
  - name: my-pull-secret
```

### Install Steps
To install Falcon Cluster Guard, run the following command to install the FalconClusterGuard CR:
```sh
kubectl create -f https://raw.githubusercontent.com/crowdstrike/falcon-operator/main/config/samples/falcon_v1alpha1_falconclusterguard.yaml --edit=true
```

To install Falcon Cluster Guard with the Image Analyzer enabled, use the following sample instead:
```sh
kubectl create -f https://raw.githubusercontent.com/crowdstrike/falcon-operator/main/config/samples/falcon_v1alpha1_falconclusterguard-with-image-analyzer.yaml --edit=true
```

### Uninstall Steps
To uninstall Falcon Cluster Guard simply remove the FalconClusterGuard resource. The operator will uninstall Falcon Cluster Guard from the cluster.

```sh
kubectl delete falconclusterguard --all
```

> [!NOTE]
> During uninstallation, the node sensor DaemonSet runs a cleanup job to remove `/opt/CrowdStrike` from each node. If cleanup pods crashloop, manually remove the `/opt/CrowdStrike` directory on affected nodes to prevent stale Agent IDs from being reused on reinstallation.

### Sensor upgrades

To upgrade the sensor version, simply add and/or update the `version` field in the FalconClusterGuard resource and apply the change. Alternatively if the `image` field was used instead of using the Falcon API credentials, add and/or update the `image` field in the FalconClusterGuard resource and apply the change. The operator will detect the change and perform the upgrade.

### Troubleshooting

- Falcon Operator modifies the FalconClusterGuard CR based on what is happening in the cluster. You can list the CR, Operator Version, and Sensor version by running the following:

  ```sh
  $ kubectl get falconclusterguard
  NAME                    OPERATOR VERSION   FALCON SENSOR
  falcon-clusterguard     0.8.0              7.33.0-1234.container.x86_64.Release.US-1
  ```

  You can get more insight by viewing the FalconClusterGuard CRD in full detail:

  ```sh
  kubectl get falconclusterguard -o yaml
  ```

- To review the logs of Falcon Operator:
  ```sh
  kubectl -n falcon-operator logs -f deploy/falcon-operator-controller-manager -c manager
  ```

- To review the logs of the admission controller:
  ```sh
  kubectl logs -n falcon-sensor -l "crowdstrike.com/provider=crowdstrike"
  ```

- To review the logs of the node sensor DaemonSet:
  ```sh
  kubectl logs -n falcon-sensor -l "crowdstrike.com/component=node_sensor"
  ```

- To review the logs of the Image Analyzer:
  ```sh
  kubectl logs -n falcon-system -l "crowdstrike.com/component=fcg-iar"
  ```

- Each enabled component reports its readiness as a status condition on the FalconClusterGuard resource (`AdmissionReady`, `NodeSensorReady`, and `ImageAnalyzerReady`). The condition is removed when its component is disabled:
  ```sh
  kubectl get falconclusterguard -o jsonpath='{range .items[].status.conditions[*]}{.type}={.status}{"\n"}{end}'
  ```

- To review the currently deployed version of the operator:
  ```sh
  kubectl get falconclusterguard -A -o=jsonpath='{.items[].status.version}'
  ```

- The admission controller process creates the following ConfigMaps at runtime (not managed by the operator):
  - `falcon-kac-aid` — stores the sensor AID (agent ID) after registration
  - `falcon-kac-meta` — stores cluster metadata (e.g. cluster name discovered by the sensor)
  - `falcon-kac-pol` — stores policy data pulled from the Falcon platform
  - `falcon-kac-state` — stores internal sensor state
