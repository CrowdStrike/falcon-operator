# Falcon Container Sensor

## About Falcon Container Sensor
The Falcon Container sensor for Linux extends runtime security to container workloads in Kubernetes clusters that don’t allow you to deploy the kernel-based Falcon sensor for Linux. The Falcon Container sensor runs as an unprivileged container in user space with no code running in the kernel of the worker node OS. This allows it to secure Kubernetes pods in clusters where it isn’t possible to deploy the kernel-based Falcon sensor for Linux on the worker node, as with AWS Fargate where organizations don’t have access to the kernel and where privileged containers are disallowed. The Falcon Container sensor can also secure container workloads on clusters where worker node security is managed separately.

### Core Features
 - **Leverage market-leading protection technologies:** Machine learning (ML), artificial intelligence (AI), indicators of attack (IOAs) and custom hash blocking automatically defend against malware and sophisticated threats targeting containers.
 - **Stop malicious behavior:** Behavioral profiling enables you to block activities that violate policy with zero impact to legitimate container operation.
 - **Investigate container incidents faster:** Easily investigate incidents when detections are associated with the specific container and not bundled with host events.
 - **See everything:** Capture container start, stop, image, runtime information and all events generated inside each and every container.
 - **Deploy seamlessly with Kubernetes:** Deploy easily at scale by including it as part of a Kubernetes cluster.
 - **Improve container orchestration:** Capture Kubernetes namespace, pod metadata, process, file and network events.

Learn more at [product pages](https://www.crowdstrike.com/products/cloud-security/falcon-cloud-workload-protection/container-security/).


## About FalconContainer Custom Resource
Falcon Operator introduces FalconContainer Custom Resource to the cluster. The resource is meant to be singleton and it will install, configure and uninstall Falcon Container Sensor on the cluster.

> [!IMPORTANT]
> To start the Falcon Container installation please push the following FalconContainer resource to your cluster. You will need to provide CrowdStrike API Keys and CrowdStrike cloud region for the installation. It is recommended to establish new API credentials for the installation at https://falcon.crowdstrike.com/support/api-clients-and-keys, minimal required permissions are:
> * Falcon Images Download: **Read**
> * Sensor Download: **Read**

No other permissions shall be granted to the new API key pair.

```yaml
apiVersion: falcon.crowdstrike.com/v1alpha1
kind: FalconContainer
metadata:
  name: falcon-sidecar-sensor
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

### FalconContainer Reference Manual

#### Falcon API Settings
| Spec                     | Description                                                                                                                                                                                                                    |
|:-------------------------|:-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| falcon_api.client_id     | (optional) CrowdStrike API Client ID                                                                                                                                                                                           |
| falcon_api.client_secret | (optional) CrowdStrike API Client Secret                                                                                                                                                                                       |
| falcon_api.cloud_region  | (optional CrowdStrike cloud region (allowed values: autodiscover, us-1, us-2, eu-1, us-gov-1, us-gov-2);<br> Falcon API credentials or [Falcon Secret with credentials](#falcon-secret-settings) are required if `cloud_region: autodiscover`;<br> `autodiscover` cannot be used for us-gov-1 or us-gov-2 |
| falcon_api.cid           | (optional) CrowdStrike Falcon CID API override; Required for us-gov-2                                                                                                                                                                                 |

#### Sidecar Injection Configuration Settings
| Spec                                      | Description                                                                                                                                                                                                             |
|:------------------------------------------|:------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| installNamespace                          | (optional) Override the default namespace of falcon-system                                                                                                                                                              |
| image                                     | (optional) Leverage a Falcon Container Sensor image that is not managed by the operator; typically used with custom repositories; overrides all registry settings; might require injector.imagePullSecretName to be set |
| version                                   | (optional) Enforce particular Falcon Container version to be installed (example: "6.31", "6.31.0", "6.31.0-1409")                                                                                                       |
| nodeAffinity                              | (optional) See https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/ for examples on configuring nodeAffinity. AMD64 and ARM64 architectures are supported by default.                               |
| registry.type                             | Registry to mirror Falcon Container (allowed values: acr, ecr, crowdstrike, gcr, openshift)                                                                                                                             |
| registry.tls.insecure_skip_verify         | (optional) Skip TLS check when pushing Falcon Container to target registry (only for demoing purposes on self-signed openshift clusters)                                                                                |
| registry.tls.caCertificate                | (optional) A string containing an optionally base64-encoded Certificate Authority Chain for self-signed TLS Registry Certificates                                                                                       |
| registry.tls.caCertificateConfigMap       | (optional) The name of a ConfigMap containing CA Certificate Authority Chains under keys ending in ".tls"  for self-signed TLS Registry Certificates (ignored when registry.tls.caCertificate is set)                   |
| registry.acr_name                         | (optional) Name of ACR for the Falcon Container push. Only applicable to Azure cloud. (`registry.type="acr"`)                                                                                                           |
| injector.serviceAccount.annotations       | (optional) Annotations that should be added to the Service Account (e.g. for IAM role association)                                                                                                                      |
| injector.listenPort                       | (optional) Override the default Injector Listen Port of 4433                                                                                                                                                            |
| injector.replicas                         | (optional) Override the default Injector Replica count of 2                                                                                                                                                             |
| injector.tls.validity                     | (optional) Override the default Injector CA validity of 3650 days                                                                                                                                                       |
| injector.imagePullPolicy                  | (optional) Override the default Falcon Container image pull policy of Always                                                                                                                                            |
| injector.imagePullSecret                  | (optional) Provide a secret containing an alternative pull token for the Falcon Container image                                                                                                                         |
| injector.logVolume                        | (optional) Provide a volume for Falcon Container logs                                                                                                                                                                   |
| injector.resources                        | (optional) Provide a set of kubernetes resource requirements for the Falcon Injector                                                                                                                                    |
| injector.sensorResources                  | (optional) Provide a set of kubernetes resource requirements for the Falcon Container Sensor container                                                                                                                  |
| injector.additionalEnvironmentVariables   | (optional) Provide additional environment variables for Falcon Container                                                                                                                                                |
| injector.disableDefaultNamespaceInjection | (optional) If set to true, disables default Falcon Container injection at the namespace scope; namespaces requiring injection will need to be labeled as specified below                                                |
| injector.disableDefaultPodInjection       | (optional) If set to true, disables default Falcon Container injection at the pod scope; pods requiring injection will need to be annotated as specified below                                                          |
| injector.alternateMountPath               | (optional) Enable volume mounts at /falcon instead of /tmp for NVCF environment                                                                                                                                         |

#### Falcon Sensor Settings
| Spec                      | Description                                                                                                                                                                                        |
|:--------------------------|:---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| falcon.cid                | (optional) CrowdStrike Falcon CID override;<br> [Falcon API credentials](#falcon-api-settings) or [Falcon Secret with credentials](#falcon-secret-settings) are required if this field is not set;<br> Required for us-gov-2 |
| falcon.apd                | (optional) Configure Falcon Sensor to leverage a proxy host                                                                                                                                        |
| falcon.aph                | (optional) Configure the host Falcon Sensor should leverage for proxying                                                                                                                           |
| falcon.app                | (optional) Configure the port Falcon Sensor should leverage for proxying                                                                                                                           |
| falcon.billing            | (optional) Configure Pay-as-You-Go (metered) billing rather than default billing                                                                                                                   |
| falcon.provisioning_token | (optional) Configure a Provisioning Token for CIDs with restricted AID provisioning enabled                                                                                                        |
| falcon.tags               | (optional) Configure Falcon Sensor Grouping Tags; comma-delimited                                                                                                                                  |
| falcon.trace              | (optional) Configure Falcon Sensor Trace Logging Level (none, err, warn, info, debug)                                                                                                              |

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

#### AITap
AITap requires configuration of the AI-DR Collector, and enabling AITap for namespaces and/or pods. The
following guide explains the different options for enabling AITap for your AI workloads.

> [!NOTE]
> AITap is only active for pods where the Falcon Container sensor is injected. Namespaces or pods not configured
> for container sensor injection will not have AITap enabled regardless of the AITap configuration.

| Parameter                                | Description                                                                                                                                                 | Default                               |
|:-----------------------------------------|:------------------------------------------------------------------------------------------------------------------------------------------------------------|:--------------------------------------|
| `injector.aitap.namespaces`              | A list of namespaces where AITap should be enabled.                                                                                                         | None                                  |
| `injector.aitap.allNamespaces`           | Enable AITap in all namespaces. Reserved system namespaces are automatically excluded.                                                                      | `false`                               |
| `injector.aitap.aidrCollectorBaseApiUrl` | AI-DR Collector Base API URL for the Application Collector                                                                                                  | None       (Required to enable AITap) |
| `injector.aitap.aidrCollectorApiToken`   | AI-DR Collector API token for the Application Collector                                                                                                     | None                                  |
| `injector.aitap.aidrSecretName`          | Custom AI-DR Kubernetes secret name                                                                                                                         | `falcon-aitap-aidr-secret`            |
| `injector.aitap.useExistingSecret`       | Use an existing AI-DR secret. When true, Operator does NOT create the AI-DR secret. When false (default), Operator propagates secrets to target namespaces. | `false`                               |

##### Managing Your Own AI-DR Secret
If you would like to manage your own AI-DR collector secret, you can use an existing secret with
the `.collector-aidr-token` data key and your AI-DR collector token as the value.

Example AI-DR secret:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: <YOUR_AIDR_SECRET_NAME>
  namespace: ai-services
type: Opaque
stringData:
  .collector-aidr-token: "<YOUR_COLLECTOR_API_TOKEN>"
```

In your FalconContainer custom resource manifest, your `injector.aitap.aidrSecretName` must match the name of the secret you created.

FalconContainer AITap Values:
```yaml
injector:
  aitap:
    useExistingSecret: true
    aidrSecretName: <YOUR_AIDR_SECRET_NAME>
    aidrCollectorBaseApiUrl: "<YOUR_COLLECTOR_API_BASE_URL>"
```

> [!NOTE]
> When `useExistingSecret` is `true`, only `aidrCollectorBaseApiUrl` is required. When not managing your own secret,
> `aidrCollectorBaseApiUrl` and `aidrCollectorApiToken` are both required.

##### Enabling AITap
AITap can be enabled for specific pods, specific namespaces, or in all namespaces.
You can control this behavior with the following options.

**Option 1: Specific Pods**

If you prefer to have granular control of AITap, you can enable AITap for specific pods only.
To enable AITap for specific pods, you must annotate the pod with `sensor.falcon-system.crowdstrike.com/enable-aitap-events: "true"`.
You must also set `useExistingSecret: true` and `aidrSecretName: <YOUR_AIDR_SECRET_NAME>`,
and create your own AI-DR secret in each applicable namespace.

Deployment manifest example with annotations:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ai-application
  namespace: production
spec:
  replicas: 3
  selector:
    matchLabels:
      app: my-application
  template:
    metadata:
      labels:
        app: my-application
      annotations:
        sensor.falcon-system.crowdstrike.com/enable-aitap-events: "true"
```

FalconContainer AITap Values:
```yaml
injector:
  aitap:
    useExistingSecret: true
    aidrSecretName: "<YOUR_AIDR_SECRET_NAME>"
    aidrCollectorBaseApiUrl: "<YOUR_COLLECTOR_API_BASE_URL>"
```

With `injector.aitap.allNamespaces: false` and `injector.aitap.namespaces` not configured:
- Falcon Container sensor does not enable AITap for any namespaces by default.
- Operator does NOT create AI-DR secrets in any namespace.
- You must create your own AI-DR secret with the `aidrSecretName` in each target namespace.

**Option 2: Specific Namespaces**
To enable AITap for specific namespaces only, use the `injector.aitap.namespaces` option.

```yaml
injector:
  aitap:
    namespaces:
      - namespace-1
      - namespace-2
      - namespace-3
    aidrCollectorApiToken: "<YOUR_COLLECTOR_API_TOKEN>"
    aidrCollectorBaseApiUrl: "<YOUR_COLLECTOR_API_BASE_URL>"
```

Once the FalconContainer custom resource is deployed, any new namespaces you want AITap enabled for must be added to
the list of `injector.aitap.namespaces` and the changes applied to your custom resource.

> [!NOTE]
> Make sure all namespaces in `injector.aitap.namespaces` already exist at the time of your FalconContainer deployment.

**Option 3: Combination of Namespaces and Pods**

You can use a combination of `namespaces` and the `sensor.falcon-system.crowdstrike.com/enable-aitap-events: "true"` pod annotation to enabled AITap for specific namespaces, and individual pods not included in the list of namespaces.

```yaml
injector:
  aitap:
    namespaces: # namespace-3 has a pod with `enable-aitap-events` annotation
      - namespace-1
      - namespace-2
    # the aidrSecretName must match the existing AI-DR secret in namespace-3.
    aidrSecretName: "<YOUR_AIDR_SECRET_NAME>"
    aidrCollectorApiToken: "<YOUR_COLLECTOR_API_TOKEN>"
    aidrCollectorBaseApiUrl: "<YOUR_COLLECTOR_API_BASE_URL>"
```

You must manage your own AI-DR secret with the same name as `aidrSecretName` in any namespace not included in the list of `namespaces`, which in this example would be 'namespace-3'.

**Option 4: Enable AITap for your entire cluster**
To enable AITap for all namespaces, use the `allNamespaces: true` option.

```yaml
injector:
  aitap:
    allNamespaces: true
    aidrCollectorApiToken: "<YOUR_COLLECTOR_API_TOKEN>"
    aidrCollectorBaseApiUrl: "<YOUR_COLLECTOR_API_BASE_URL>"
```

AITap will be enabled in:
- All other namespaces **except** reserved system namespaces

The following namespaces are automatically excluded:
- `kube-system`, `kube-public`, `kube-node-lease`
- `falcon-system`, `falcon-kac`, `falcon-iar`
- The deployment namespace

Once the FalconContainer custom resource is deployed, AITap will not instantly be enabled for new namespaces
created after the initial deployment.

#### Advanced Settings
The following settings provide an alternative means to select which version of Falcon sensor is deployed. Their use is not recommended. Instead, an explicit SHA256 hash should be configured using the `image` property above.

See `docs/ADVANCED.md` for more details.

| Spec | Default Value | Description |
| :- | :- | :- |
| advanced.autoUpdate | `off` | Automatically updates a deployed Falcon sensor as new versions are released. This has no effect if a specific image or version has been requested. Valid settings are:<ul><li>`force` -- Reconciles the resource after every check for a new version</li><li>`normal` -- Reconciles the resource whenever a new version is detected</li><li>`off` -- No automatic updates</li></ul>
| advanced.updatePolicy | _none_ | If set, applies the named Linux sensor update policy, configured in Falcon UI, to select which version of Falcon sensor to install. The policy must be enabled and must match the CPU architecture of the cluster (AMD64 or ARM64). |

> [!NOTE]
> Falcon Container sensor for Linux does not support the **Uninstall and maintenance protection** policy setting and
> automatically ignores it.

##### Automatic Update Frequency
The operator checks for new releases of Falcon sensor once every 24 hours by default. This can be adjusted by setting the `--sensor-auto-update-interval` command-line flag to any value acceptable by [Golang's ParseDuration](https://pkg.go.dev/time#ParseDuration) function. However, it is strongly recommended that this be left at the default, as each cycle involves queries to the Falcon API and too many could result in throttling.

#### Status Conditions
| Status                              | Description                                                                                                                               |
| :---------------------------------- | :---------------------------------------------------------------------------------------------------------------------------------------- |
| conditions.["NamespaceReady"]                    | Displays the most recent reconciliation operation for the Namespace used by the Falcon Container Sensor (Created, Updated, Deleted)                                  |
| conditions.["ImageReady"]                        | Informs about readiness of Falcon Container image. Custom message refers to image URI that will be used during the deployment (Pushed, Discovered)                   |
| conditions.["ImageStreamReady"]                  | Displays the most recent successful reconciliation operation for the image stream used by the falcon container in openshift environments (created, updated, deleted) |
| conditions.["ServiceAccountReady"]               | Displays the most recent successful reconciliation operation for the service account used by the falcon container (created, updated, deleted)                                   |
| conditions.["ClusterRoleReady"]                  | Displays the most recent successful reconciliation operation for the cluster role used by the falcon container sensor (created, updated, deleted)                               |
| conditions.["ClusterRoleBindingReady"]           | Displays the most recent successful reconciliation operation for the cluster role binding used by the falcon container sensor (created, updated, deleted)                       |
| conditions.["SecretReady"]                       | Displays the most recent successful reconciliation operation for the secrets used by the falcon container sensor (created, updated, deleted)                                    |
| conditions.["ConfigMapReady"]                    | Displays the most recent successful reconciliation operation for the config map used by the falcon container sensor (created, updated, deleted)                                 |
| conditions.["DeploymentReady"]                   | Displays the most recent successful reconciliation operation for the deployment used by the falcon container sensor injector (created, updated, deleted)                        |
| conditions.["ServiceReady"]                      | Displays the most recent successful reconciliation operation for the service used by the falcon container sensor injector (created, updated, deleted)                           |
| conditions.["MutatingWebhookConfigurationReady"] | Displays the most recent successful reconciliation operation for the mutating webhook configuration used by the falcon container sensor injector (created, updated, deleted)    |

> [!IMPORTANT]
> All arguments are optional, but successful deployment requires either **client_id and client_secret or the Falcon cid and image**. When deploying using the CrowdStrike Falcon API, the container image and CID will be fetched from CrowdStrike Falcon API. While in the latter case, the CID and image location is explicitly specified by the user.

### Enabling and Disabling Falcon Container injection

By default, all pods in all namespaces outside of `kube-system` and `kube-public` will be subject to Falcon Container injection.

To disable sensor injection for all pods in one namespace, add a label to the namespace:
```yaml
sensor.falcon-system.crowdstrike.com/injection=disabled
```

If `injector.disableDefaultNamespaceInjection` is set to `true`, then sensor injection will be disabled in all namespaces by default. To enable injection for all pods in one namespace with default namespace injection set to `true`, add a label to the namespace:
```yaml
sensor.falcon-system.crowdstrike.com/injection=enabled
```

To disable sensor injection for one pod, add an annotation to the pod spec:
```yaml
sensor.falcon-system.crowdstrike.com/injection=disabled
```

If `injector.disableDefaultPodInjection` is set to `true`, then sensor injection will be disabled for all pods by default. To enable injection for one pod in a namespace subject to injection, add an annotation to the pod spec:
```yaml
sensor.falcon-system.crowdstrike.com/injection=enabled
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

Falcon Container Image is distributed by CrowdStrike through CrowdStrike Falcon registry. Operator supports two modes of deployment:

#### (Option 1) Use CrowdStrike registry directly

Does not require any advanced set-ups. Users are advised to use the following excerpt in theirs FalconContainer custom resource definition.

```yaml
registry:
  type: crowdstrike
```

Falcon Container product will then be installed directly from CrowdStrike registry. Any new deployment to the cluster may contact CrowdStrike registry for the image download. The `falcon-crowdstrike-pull-secret imagePullSecret` is created in all the namespaces targeted for injection.

#### (Option 2) Let operator mirror Falcon Container image to your local registry

Requires advanced set-up to grant the operator push access to your local registry. The operator will then mirror Falcon Container image from CrowdStrike registry to your local registry of choice.
Supported registries are: acr, ecr, gcr, and openshift. Each registry type requires advanced set-up enable image push.

Consult specific deployment guides to learn about the steps needed for image mirroring.

- [Deployment Guide for OpenShift](../../README.md)

#### (Option 3) Use a custom Image URI

Image must be available at the specified URI; setting the image attribute will cause registry settings to be ignored. No image mirroring will be leveraged.

Example:
```yaml
image: myprivateregistry.internal.lan/falcon-container/falcon-sensor:6.47.0-3003.container.x86_64.Release.US-1
```

### Install Steps
To install Falcon Container (assuming Falcon Operator is installed):
```sh
oc create -f https://raw.githubusercontent.com/crowdstrike/falcon-operator/main/config/samples/falcon_v1alpha1_falconcontainer.yaml --edit=true
```

### Uninstall Steps
To uninstall Falcon Container simply remove the FalconContainer resource. The operator will uninstall the Falcon Container product from the cluster.

```sh
oc delete falconcontainers.falcon.crowdstrike.com --all
```

### Namespace Reference

The following namespaces will be used by Falcon Operator.

| Namespace               | Description                                                      |
|:------------------------|:-----------------------------------------------------------------|
| falcon-system           | Used by Falcon Container product, runs the injector, and webhoook |
| falcon-operator         | Runs falcon-operator manager                                      |

### Sensor upgrades

To upgrade the sensor version, simply add and/or update the `version` field in the FalconContainer resource and apply the change. Alternatively if the `image` field was used instead of using the Falcon API credentials, add and/or update the `image` field in the FalconContainer resource and apply the change. The operator will detect the change and perform the upgrade.

> [!IMPORTANT]
> The operator will only upgrade the injector service. You will need to restart or roll your workload deployments to upgrade the sidecar version.

### Troubleshooting

- Falcon Operator modifies the FalconContainer CR based on what is happening in the cluster. You can get list the CR, Operator Version, and Sensor version by running the following:

  ```sh
  $ oc get falconcontainers.falcon.crowdstrike.com
  NAME                    OPERATOR VERSION   FALCON SENSOR
  falcon-sidecar-sensor   0.8.0              6.51.0-3401.container.x86_64.Release.US-1
  ```

  This is helpful information to use as a starting point for troubleshooting.
  You can get more insight by viewing the FalconContainer CRD in full detail by running the following command:

  ```sh
  oc get falconcontainers.falcon.crowdstrike.com -o yaml
  ```

- To review the logs of Falcon Operator:
  ```sh
  oc -n falcon-operator logs -f deploy/falcon-operator-controller-manager -c manager
  ```

- To review the logs of Falcon Container Sidecar Injector service:
  ```sh
  oc logs -n falcon-system -l "crowdstrike.com/provider=crowdstrike"
  ```

- To review the currently deployed version of the operator:
  ```sh
  oc get falconnodesensors -A -o=jsonpath='{.items[].status.version}'
  ```


### Additional Documentation
End-to-end guide(s) to install Falcon-operator together with FalconContainer resource.
 - [Deployment Guide for OpenShift](../../README.md)
