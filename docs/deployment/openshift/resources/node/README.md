# Falcon Node Sensor

## About Falcon Cloud Workload Protection

Learn more at [product page](https://www.crowdstrike.com/cloud-security-products/falcon-cloud-workload-protection/) and [Linux sensor blog](https://www.crowdstrike.com/blog/tech-center/linux-protection/).

## About FalconNodeSensor Custom Resource (CR)
Falcon Operator introduces the FalconNodeSensor Custom Resource (CR) to the cluster. The resource is meant to install, configure, and uninstall the Falcon Linux Sensor on the cluster nodes. This resource deploys a kernel module to the Kubernetes nodes which runs as _privileged_.

### FalconNodeSensor CR Configuration using CrowdStrike API Keys
To start the FalconNodeSensor installation using CrowdStrike API Keys to allow the operator to determine your Falcon Customer ID (CID) as well as pull down the CrowdStrike Falcon Sensor container image, please create the following FalconNodeSensor resource to your cluster. You will need to provide CrowdStrike API Keys and CrowdStrike cloud region for the installation. It is recommended to establish new API credentials for the installation at https://falcon.crowdstrike.com/support/api-clients-and-keys, required permissions are:
 * Falcon Images Download: Read
 * Sensor Download: Read

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
| node.tolerations                    | (optional) See https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/ for examples on configuring tolerations      |
| node.nodeAffinity                   | (optional) See https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/ for examples on configuring nodeAffinity          |
| node.image                          | (optional) Location of the Falcon Sensor Image. Specify only when you mirror the original image to your own image repository              |
| node.imagePullPolicy                | (optional) Override the default Falcon Container image pull policy of Always                                                              |
| node.imagePullSecrets               | (optional) list of references to secrets to use for pulling image from image_override location.                                           |
| node.terminationGracePeriod         | (optional) Kills pod after a specificed amount of time (in seconds). Default is 30 seconds.                                               |
| node.serviceAccount.annotations     | (optional) Annotations that should be added to the Service Account (e.g. for IAM role association)                                        |
| node.disableCleanup                 | (optional) Cleans up `/opt/CrowdStrike` on the nodes by deleting the files and directory.                                                 |
| node.version                        | (optional) Enforce particular Falcon Sensor version to be installed (example: "6.35", "6.35.0-13207")                                     |

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

All arguments are optional, but successful deployment requires either falcon_id and falcon_secret **or** cid and image. When deploying using the CrowdStrike Falcon API, the container image and CID will be fetched from CrowdStrike Falcon API. While in the latter case, the CID and image location is explicitly specified by the user.

### Install Steps
With Falcon Operator installed, run the following command to install the FalconNodeSensor CR:
```sh
oc create -f https://raw.githubusercontent.com/CrowdStrike/falcon-operator/main/config/samples/falcon_v1alpha1_falconnodesensor.yaml --edit=true
```
The above command uses an example `yaml` file from the Falcon Operator GitHub repository that allows you to easily configure the FalconNodeSensor CR using the Falcon API method.

### Uninstall Steps
To uninstall the FalconNodeSensor CR, simply remove the FalconNodeSensor resource. The operator will uninstall the Falcon Sensor from the cluster.

```sh
oc delete falconnodesensors --all
```

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
