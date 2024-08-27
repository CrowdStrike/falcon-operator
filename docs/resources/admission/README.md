# Falcon Admission Controller

## About FalconAdmission Custom Resource (CR)
Falcon Operator introduces the FalconAdmission Custom Resource (CR) to the cluster. The resource is meant to install, configure, and uninstall the Falcon Admission Controller on the cluster.

### FalconAdmission CR Configuration using CrowdStrike API Keys
To start the FalconAdmission installation using CrowdStrike API Keys to allow the operator to determine your Falcon Customer ID (CID) as well as pull down the CrowdStrike Falcon Admission Controller image, please create the following FalconAdmission resource to your cluster.

> [!IMPORTANT]
> You will need to provide CrowdStrike API Keys and CrowdStrike cloud region for the installation. It is recommended to establish new API credentials for the installation at https://falcon.crowdstrike.com/support/api-clients-and-keys, required permissions are:
> * Falcon Images Download: **Read**
> * Sensor Download: **Read**

Example:

```yaml
apiVersion: falcon.crowdstrike.com/v1alpha1
kind: FalconAdmission
metadata:
  name: falcon-admission
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

### FalconAdmission Reference Manual

#### Falcon API Settings
| Spec                       | Description                                                                                              |
| :------------------------- | :------------------------------------------------------------------------------------------------------- |
| falcon_api.client_id       | CrowdStrike API Client ID                                                                                |
| falcon_api.client_secret   | CrowdStrike API Client Secret                                                                            |
| falcon_api.cloud_region    | CrowdStrike cloud region (allowed values: autodiscover, us-1, us-2, eu-1, us-gov-1)                      |
| falcon_api.cid             | (optional) CrowdStrike Falcon CID API override                                                           |

#### Admission Controller Configuration Settings
| Spec                                      | Description                                                                                                                                                                                                             |
| :----------------------------------       | :----------------------------------------------------------------------------------------------------------------------------------------                                                                               |
| installNamespace                          | (optional) Override the default namespace of falcon-kac                                                                                                                                                                 |
| image                                     | (optional) Leverage a Falcon Admission Controller Sensor image that is not managed by the operator; typically used with custom repositories; overrides all registry settings; might require admissionConfig.imagePullSecrets to be set |
| version                                   | (optional) Enforce particular Falcon Admission Controller version to be installed (example: "6.31", "6.31.0", "6.31.0-1409")                                                                                            |
| registry.type                             | Registry to mirror Falcon Admission Controller (allowed values: acr, ecr, crowdstrike, gcr, openshift)                                                                                                                  |
| registry.tls.insecure_skip_verify         | (optional) Skip TLS check when pushing Falcon Admission to target registry (only for demoing purposes on self-signed openshift clusters)                                                                                |
| registry.tls.caCertificate                | (optional) A string containing an optionally base64-encoded Certificate Authority Chain for self-signed TLS Registry Certificates                                                                                       |
| registry.tls.caCertificateConfigMap       | (optional) The name of a ConfigMap containing CA Certificate Authority Chains under keys ending in ".tls"  for self-signed TLS Registry Certificates (ignored when registry.tls.caCertificate is set)                   | 
| registry.acr_name                         | (optional) Name of ACR for the Falcon Admission push. Only applicable to Azure cloud. (`registry.type="acr"`)                                                                                                           |
| resourcequota.pods                        | (optional) Configure the maximum number of pods that can be created in the falcon-kac namespace                                                                                                                         |
| admissionConfig.serviceAccount.annotations| (optional) Configure annotations for the falcon-kac service account (e.g. for IAM role association)                                                                                                                    |
| admissionConfig.servicePort               | (optional) Configure the port the Falcon Admission Controller Service listens on                                                                                                                                        |
| admissionConfig.containerPort             | (optional) Configure the port the Falcon Admission Controller container listens on                                                                                                                                       |
| admissionConfig.tls.validity              | (optional) Configure the validity of the TLS certificate used by the Falcon Admission Controller                                                                                                                         |
| admissionConfig.failurePolicy             | (optional) Configure the failure policy of the Falcon Admission Controller                                                                                                                                             |
| admissionConfig.disabledNamespaces.namespaces                | (optional) Configure the list of namespaces the Falcon Admission Controller validating webhook should ignore                                                                                         |
| admissionConfig.snapshotsEnabled          | (optional) Determines if snapshots of Kubernetes resources are periodically taken for cluster visibility.                                                                                                                |
| admissionConfig.snapshotsInterval         | (optional) Time interval between two snapshots of Kubernetes resources in the cluster                                                                                                                                    |
| admissionConfig.watcherEnabled            | (optional) Determines if Kubernetes resources are watched for cluster visibility                                                                                                                                         |
| admissionConfig.replicas                  | (optional) Currently ignored and internally set to 1                                                                                                                                                                    |
| admissionConfig.imagePullPolicy           | (optional) Configure the image pull policy of the Falcon Admission Controller                                                                                                                                           |
| admissionConfig.imagePullSecrets          | (optional) Configure the image pull secrets of the Falcon Admission Controller                                                                                                                                          |
| admissionConfig.resourcesClient           | (optional) Configure the resources client of the Falcon Admission Controller                                                                                                                                            |
| admissionConfig.resourcesWatcher          | (optional) Configure the resources watcher of the Falcon Admission Controller                                                                                                                                            |
| admissionConfig.resources                 | (optional) Configure the resources of the Falcon Admission Controller                                                                                                                                                  |
| admissionConfig.updateStrategy            | (optional) Configure the deployment update strategy of the Falcon Admission Controller                                                                                                                                  |


#### Falcon Sensor Settings
| Spec                                      | Description                                                                                                                                                                                                             |
| :----------------------------------       | :----------------------------------------------------------------------------------------------------------------------------------------                                                                               |
| falcon.apd                                | (optional) Configure Falcon Sensor to leverage a proxy host                                                                                                                                                             |
| falcon.aph                                | (optional) Configure the host Falcon Sensor should leverage for proxying                                                                                                                                                |
| falcon.app                                | (optional) Configure the port Falcon Sensor should leverage for proxying                                                                                                                                                |
| falcon.billing                            | (optional) Configure Pay-as-You-Go (metered) billing rather than default billing                                                                                                                                        |
| falcon.provisioning_token                 | (optional) Configure a Provisioning Token for CIDs with restricted AID provisioning enabled                                                                                                                             |
| falcon.tags                               | (optional) Configure Falcon Sensor Grouping Tags; comma-delimited                                                                                                                                                       |
| falcon.trace                              | (optional) Configure Falcon Sensor Trace Logging Level (none, err, warn, info, debug)                                                                                                                                   |

> [!IMPORTANT]
> All arguments are optional, but successful deployment requires either **client_id and client_secret or the Falcon cid and image**. When deploying using the CrowdStrike Falcon API, the container image and CID will be fetched from CrowdStrike Falcon API. While in the latter case, the CID and image location is explicitly specified by the user.

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

Falcon Admission Image is distributed by CrowdStrike through CrowdStrike Falcon registry. Operator supports two modes of deployment:

#### (Option 1) Use CrowdStrike registry directly

Does not require any advanced setup. Users are advised to use the following except in their FalconAdmission custom resource definition.

```yaml
registry:
  type: crowdstrike
```

Falcon Admission product will then be installed directly from CrowdStrike registry. Any new deployment to the cluster may contact CrowdStrike registry for the image download.

#### (Option 2) Let operator mirror Falcon Admission Controller image to your local registry

Requires advanced setup to grant the operator push access to your local registry. The operator will then mirror the Falcon Admission image from CrowdStrike registry to your local registry of choice.
Supported registries are: acr, ecr, gcr, and openshift. Each registry type requires advanced setup enable image push.

Consult specific deployment guides to learn about the steps needed for image mirroring.

 - [Deployment Guide for AKS/ACR](../../deployment/azure/README.md)
 - [Deployment Guide for EKS/ECR](../../deployment/eks/README.md)
 - [Deployment Guide for EKS Fargate](../../deployment/eks-fargate/README.md)
 - [Deployment Guide for GKE/GCR](../../deployment/gke/README.md)
 - [Deployment Guide for OpenShift](../../deployment/openshift/README.md)

#### (Option 3) Use a custom Image URI

Image must be available at the specified URI; setting the image attribute will cause registry settings to be ignored. No image mirroring will be leveraged.

Example:
```yaml
image: myprivateregistry.internal.lan/falcon-admission/falcon-sensor:6.47.0-3003.container.x86_64.Release.US-1
```

### Install Steps
To install Falcon Admission Controller, run the following command to install the FalconAdmission CR:
```sh
kubectl create -f https://raw.githubusercontent.com/crowdstrike/falcon-operator/main/config/samples/falcon_v1alpha1_falconadmission.yaml --edit=true
```

### Uninstall Steps
To uninstall Falcon Admission Controller simply remove the FalconAdmission resource. The operator will uninstall the Falcon Admission Controller from the cluster.

```sh
kubectl delete falconadmission --all
``` 

### Sensor upgrades

To upgrade the sensor version, simply add and/or update the `version` field in the FalconAdmission resource and apply the change. Alternatively if the `image` field was used instead of using the Falcon API credentials, add and/or update the `image` field in the FalconAdmission resource and apply the change. The operator will detect the change and perform the upgrade.

### Troubleshooting

- Falcon Operator modifies the FalconAdmission CR based on what is happening in the cluster. You can get list the CR, Operator Version, and Sensor version by running the following:

  ```sh
  $ kubectl get falconadmission
  NAME                    OPERATOR VERSION   FALCON SENSOR
  falcon-admission        0.8.0              6.51.0-3401.container.x86_64.Release.US-1
  ```

  This is helpful information to use as a starting point for troubleshooting.  
  You can get more insight by viewing the FalconAdmission CRD in full detail by running the following command:

  ```sh
  kubectl get falconadmission -o yaml
  ```

- To review the logs of Falcon Operator:
  ```sh
  kubectl -n falcon-operator logs -f deploy/falcon-operator-controller-manager -c manager
  ```

- To review the logs of Falcon Admission controller service:
  ```sh
  kubectl logs -n falcon-kac -l "crowdstrike.com/provider=crowdstrike"
  ```

- To review the currently deployed version of the operator:
  ```sh
  kubectl get falconadmission -A -o=jsonpath='{.items[].status.version}'
  ```


### Additional Documentation
End-to-end guide(s) to install Falcon-operator together with FalconAdmission resource.
 - [Deployment Guide for AKS/ACR](../../deployment/azure/README.md)
 - [Deployment Guide for EKS/ECR](../../deployment/eks/README.md)
 - [Deployment Guide for EKS Fargate](../../deployment/eks-fargate/README.md)
 - [Deployment Guide for GKE/GCR](../../deployment/gke/README.md)
 - [Deployment Guide for OpenShift](../../deployment/openshift/README.md)



