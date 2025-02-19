# Falcon Node Sensor

## About Falcon Cloud Workload Protection

Learn more at [product page](https://www.crowdstrike.com/cloud-security-products/falcon-cloud-workload-protection/) and [Linux sensor blog](https://www.crowdstrike.com/blog/tech-center/linux-protection/).

## About FalconNodeSensor Custom Resource (CR)
Falcon Operator introduces the FalconNodeSensor Custom Resource (CR) to the cluster. The resource is meant to install, configure, and uninstall the Falcon Linux Sensor on the cluster nodes. This resource deploys a kernel module to the Kubernetes nodes which runs as _privileged_.

### FalconNodeSensor CR Configuration using CrowdStrike API Keys

> [!IMPORTANT]
> To start the FalconNodeSensor installation using CrowdStrike API Keys to allow the operator to determine your Falcon Customer ID (CID) as well as pull down the CrowdStrike Falcon Sensor container image, please create the following FalconNodeSensor resource to your cluster. You will need to provide CrowdStrike API Keys and CrowdStrike cloud region for the installation. It is recommended to establish new API credentials for the installation at https://falcon.crowdstrike.com/support/api-clients-and-keys, required permissions are:
> * Falcon Images Download: **Read**
> * Sensor Download: **Read**

Example:
```yaml
apiVersion: falcon.crowdstrike.com/v1alpha1
kind: FalconNodeSensor
metadata:
  name: falcon-node-sensor
spec:
  falcon_api:
    client_id: PLEASE_FILL_IN
    client_secret: PLEASE_FILL_IN
    cloud_region: autodiscover
  node: {}
  falcon: {}
```

### FalconNodeSensor CR Configuration with Falcon Customer ID (CID) and non-CrowdStrike Registry

Example:
```yaml
apiVersion: falcon.crowdstrike.com/v1alpha1
kind: FalconNodeSensor
metadata:
  name: falcon-node-sensor
spec:
  falcon:
    cid: PLEASE_FILL_IN
  node:
    image: myregistry/project/image:version
```

### FalconNodeSensor Reference Manual

#### Falcon API Settings
| Spec                                | Description                                                                                                                               |
| :---------------------------------- | :---------------------------------------------------------------------------------------------------------------------------------------- |
| falcon_api.client_id                | (optional) CrowdStrike API Client ID                                                                                                      |
| falcon_api.client_secret            | (optional) CrowdStrike API Client Secret                                                                                                  |
| falcon_api.cloud_region             | (optional) CrowdStrike cloud region (allowed values: autodiscover, us-1, us-2, eu-1, us-gov-1)                                            |
| falcon_api.cid                      | (optional) CrowdStrike Falcon CID API override                                                                                            |

#### Node Configuration Settings
| Spec                                | Description                                                                                                                               |
| :---------------------------------- | :---------------------------------------------------------------------------------------------------------------------------------------- |
| installNamespace                    | (optional) Override the default namespace of falcon-system                                                                                |
| node.tolerations                    | (optional) See https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/ for examples on configuring tolerations      |
| node.nodeAffinity                   | (optional) See https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/ for examples on configuring nodeAffinity          |
| node.image                          | (optional) Location of the Falcon Sensor Image. Specify only when you mirror the original image to your own image repository              |
| node.imagePullPolicy                | (optional) Override the default Falcon Container image pull policy of Always                                                              |
| node.imagePullSecrets               | (optional) list of references to secrets to use for pulling image from image_override location.                                           |
| node.terminationGracePeriod         | (optional) Kills pod after a specified amount of time (in seconds). Default is 30 seconds.                                                |
| node.serviceAccount.annotations     | (optional) Annotations that should be added to the Service Account (e.g. for IAM role association)                                        |
| node.backend                        | (optional) Configure the backend mode for Falcon Sensor (allowed values: kernel, bpf)                                                     |
| node.disableCleanup                 | (optional) Cleans up `/opt/CrowdStrike` on the nodes by deleting the files and directory.                                                 |
| node.version                        | (optional) Enforce particular Falcon Sensor version to be installed (example: "6.35", "6.35.0-13207")                                     |

> [!IMPORTANT]
> node.tolerations will be appended to the existing tolerations for the daemonset due to GKE Autopilot allowing users to manage Tolerations directly in the console. See documentation here: https://cloud.google.com/kubernetes-engine/docs/how-to/workload-separation. Removing Tolerations from an existing daemonset requires a redeploy of the FalconNodeSensor manifest.

#### Falcon Sensor Settings
| Spec                                | Description                                                                                                                                                                |
| :---------------------------------- | :------------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| falcon.cid                          | (optional) CrowdStrike Falcon CID override                                                                                                                                 |
|	falcon.apd                          | (optional) Disable the Falcon Sensor's use of a proxy.                                                                                                                     |
|	falcon.aph                          | (optional)  The application proxy host to use for Falcon sensor proxy configuration.                                                                                       |
|	falcon.app                          | (optional)  The application proxy port to use for Falcon sensor proxy configuration.                                                                                       |
|	falcon.billing                      | (optional)  Utilize default or Pay-As-You-Go billing.                                                                                                                      |
|	falcon.provisioning_token           | (optional)  Installation token that prevents unauthorized hosts from being accidentally or maliciously added to your customer ID (CID).                                    |
|	falcon.tags                         | (optional)  Sensor grouping tags are optional, user-defined identifiers that can used to group and filter hosts. Allowed characters: all alphanumerics, '/', '-', and '_'. |
|	falcon.trace                        | (optional)  Set sensor trace level.                                                                                                                                        |

#### Advanced Settings
The following settings provide an alternative means to select which version of Falcon sensor is deployed. Their use is not recommended. Instead, an explicit SHA256 hash should be configured using the `node.image` property above.

See `docs/ADVANCED.md` for more details.

| Spec | Default Value | Description |
| :- | :- | :- |
| node.advanced.autoUpdate | `off` | Automatically updates a deployed Falcon sensor as new versions are released. This has no effect if a specific image or version has been requested. Valid settings are:<ul><li>`force` -- Reconciles the resource after every check for a new version</li><li>`normal` -- Reconciles the resource whenever a new version is detected</li><li>`off` -- No automatic updates</li></ul>
| node.advanced.updatePolicy | _none_ | If set, applies the named Linux sensor update policy, configured in Falcon UI, to select which version of Falcon sensor to install. The policy must be enabled and must match the CPU architecture of the cluster (AMD64 or ARM64). |

##### Automatic Update Frequency
The operator checks for new releases of Falcon sensor once every 24 hours by default. This can be adjusted by setting the `--sensor-auto-update-interval` command-line flag to any value acceptable by [Golang's ParseDuration](https://pkg.go.dev/time#ParseDuration) function. However, it is strongly recommended that this be left at the default, as each cycle involves queries to the Falcon API and too many could result in throttling.

> [!IMPORTANT]
> All arguments are optional, but successful deployment requires either **client_id and falcon_secret or the Falcon cid and image**. When deploying using the CrowdStrike Falcon API, the container image and CID will be fetched from CrowdStrike Falcon API. While in the latter case, the CID and image location is explicitly specified by the user.

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


### Install Steps
With Falcon Operator installed, run the following command to install the FalconNodeSensor CR:
```sh
oc create -f https://raw.githubusercontent.com/crowdstrike/falcon-operator/main/config/samples/falcon_v1alpha1_falconnodesensor.yaml --edit=true
```
The above command uses an example `yaml` file from the Falcon Operator GitHub repository that allows you to easily configure the FalconNodeSensor CR using the Falcon API method.

### Uninstall Steps
To uninstall the FalconNodeSensor CR, simply remove the FalconNodeSensor resource. The operator will uninstall the Falcon Sensor from the cluster.

```sh
oc delete falconnodesensors --all
```

### Sensor upgrades

To upgrade the sensor version, simply add and/or update the `version` field in the FalconNodeSensor resource and apply the change. Alternatively if the `image` field was used instead of using the Falcon API credentials, add and/or update the `image` field in the FalconNodeSensor resource and apply the change. The operator will detect the change and perform the upgrade by restarting the daemonset pods one by one.

### Troubleshooting

- To see the FalconNodeSensor resource on the cluster which includes the operator and sensor versions:
  ```sh
  oc get falconnodesensors -A
  ```

- To verify the existence of the daemonset object:
  ```sh
  oc get daemonsets.apps -n mynamespace
  ```
  where `mynamespace` is the installed namespace of FalconNodeSensor.

- To verify the existence of the sensor objects:
  ```sh
  oc get pods -n mynamespace
  ```
  where `mynamespace` is the installed namespace of FalconNodeSensor.

- To review the logs of Falcon Operator:
  ```sh
  oc -n falcon-operator logs -f deploy/falcon-operator-controller-manager -c manager
  ```

- To review the currently deployed version of the operator:
  ```sh
  oc get falconnodesensors -A -o=jsonpath='{.items[].status.version}'
  ```
