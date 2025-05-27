# Falcon Image Analyzer

## About FalconImageAnalyzer Custom Resource (CR)
Falcon Operator introduces the FalconImageAnalyzer Custom Resource (CR) to the cluster. The resource is meant to install, configure, and uninstall the Falcon Image Analyzer on the cluster.

### FalconImageAnalyzer CR Configuration using CrowdStrike API Keys
To start the FalconImageAnalyzer installation using CrowdStrike API Keys to allow the operator to determine your Falcon Customer ID (CID) as well as pull down the CrowdStrike Falcon Image Analyzer image, please create the following FalconImageAnalyzer resource to your cluster.

> [!IMPORTANT]
> You will need to provide CrowdStrike API Keys and CrowdStrike cloud region for the installation. It is recommended to establish new API credentials for the installation at https://falcon.crowdstrike.com/support/api-clients-and-keys, required permissions are:
> * Falcon Container CLI: **Write**
> * Falcon Container Image: **Read/Write**
> * Falcon Images Download: **Read**

Example:

```yaml
apiVersion: falcon.crowdstrike.com/v1alpha1
kind: FalconImageAnalyzer
metadata:
  name: falcon-image-analyzer
spec:
  falcon_api:
    client_id: PLEASE_FILL_IN
    client_secret: PLEASE_FILL_IN
    cloud_region: us-1
  registry:
    type: crowdstrike
```

### FalconImageAnalyzer Reference Manual

#### Falcon API Settings
| Spec                     | Description                                                                                                                                                                                                                          |
|:-------------------------|:-------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------|
| falcon_api.client_id     | (optional) CrowdStrike API Client ID                                                                                                                                                                                                 |
| falcon_api.client_secret | (optional) CrowdStrike API Client Secret                                                                                                                                                                                             |
| falcon_api.cloud_region  | (optional) CrowdStrike cloud region (allowed values: autodiscover, us-1, us-2, eu-1, us-gov-1);<br> Falcon API credentials or [Falcon Secret with credentials](#falcon-secret-settings) are required if `cloud_region: autodiscover` |
| falcon_api.cid           | (optional) CrowdStrike Falcon CID API override                                                                                                                                                                                       |

#### Falcon Image Analyzer Configuration Settings
| Spec                                      | Description                                                                                                                                                                                                             |
| :----------------------------------       | :----------------------------------------------------------------------------------------------------------------------------------------                                                                               |
| installNamespace                          | (optional) Override the default namespace of falcon-iar                                                                                                                                                                 |
| image                                     | (optional) Leverage a Falcon Image Analyzer Sensor image that is not managed by the operator; typically used with custom repositories; overrides all registry settings; might require imageAnalyzerConfig.imagePullSecrets to be set |
| version                                   | (optional) Enforce particular Falcon Image Analyzer version to be installed (example: "6.31", "6.31.0", "6.31.0-1409")                                                                                            |
| registry.type                             | Registry to mirror Falcon Image Analyzer (allowed values: acr, ecr, crowdstrike, gcr, openshift)                                                                                                                  |
| registry.tls.insecure_skip_verify         | (optional) Skip TLS check when pushing Falcon Image Analyzer to target registry (only for demoing purposes on self-signed openshift clusters)                                                                           |
| registry.tls.caCertificate                | (optional) A string containing an optionally base64-encoded Certificate Authority Chain for self-signed TLS Registry Certificates                                                                                       |
| registry.tls.caCertificateConfigMap       | (optional) The name of a ConfigMap containing CA Certificate Authority Chains under keys ending in ".tls"  for self-signed TLS Registry Certificates (ignored when registry.tls.caCertificate is set)                   |
| registry.acr_name                         | (optional) Name of ACR for the Falcon Falcon Image Analyzer push. Only applicable to Azure cloud. (`registry.type="acr"`)                                                                                               |
| imageAnalyzerConfig.serviceAccount.annotations | (optional) Configure annotations for the falcon-iar service account (e.g. for IAM role association)                                                                                                                |
| imageAnalyzerConfig.azureConfigPath       | (optional) Azure  config file path                                                                                                                                        |
| imageAnalyzerConfig.sizeLimit             | (optional) Configure the size limit of the temp storage space for scanning. By Default, this is set to `20Gi`.                                                                                                          |
| imageAnalyzerConfig.mountPath             | (optional) Configure the location of the temp storage space for scanning. By Default, this is set to `/tmp`.                                                                                                            |
| imageAnalyzerConfig.clusterName           | (required) K8s cluster name                                                                                                                                            |
| imageAnalyzerConfig.debug                 | (optional) Set to `true` for debug level log                                                                                        |
| imageAnalyzerConfig.priorityClass.name        | (optional) Set to avoid pod evictions due to resource limits.                                                                                                                                           |
| imageAnalyzerConfig.exclusions.registries     | (optional) Set the value as a list of registries to be excluded. All images in that registry(s) will be excluded                                                                                                    |
| imageAnalyzerConfig.exclusions.namespaces     | (optional) Set the value as a list of namespaces to be excluded. All pods in that namespace(s) will be excluded                                                                                                     |
| imageAnalyzerConfig.imagePullPolicy           | (optional) Configure the image pull policy of the Falcon Image Analyzer                                                                                                                                           |
| imageAnalyzerConfig.imagePullSecrets          | (optional) Configure the image pull secrets of the Falcon Image Analyzer                                                                                                                                          |
| imageAnalyzerConfig.registryConfig.credentials | (optional) Use this to provide registry secrets in the form of a list of maps. e.g.<pre>- namespace: ns1<br>&nbsp;&nbsp;secretName: mysecretname</pre>To scan OpenShift control plane components, specify the cluster's pull secret: <pre>- namespace: openshift-config<br>&nbsp;&nbsp;secretName: pull-secret</pre>  |
| imageAnalyzerConfig.resources                 | (optional) Configure the resources of the Falcon Image Analyzer                                                                                                                                                  |
| imageAnalyzerConfig.updateStrategy            | (optional) Configure the deployment update strategy of the Falcon Image Analyzer                                                                                                                                  |

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
| Secret Key                | Description                                                                                                     |
|:--------------------------|:----------------------------------------------------------------------------------------------------------------|
| falcon-client-id          | Replaces [`falcon_api.client_id`](#falcon-api-settings); Requires `falcon_api.cloud` in CRD spec is defined     |
| falcon-client-secret      | Replaces [`falcon_api.client_secret`](#falcon-api-settings); Requires `falcon_api.cloud` in CRD spec is defined |
| falcon-cid                | Replaces [`falcon_api.cid`](#falcon-api-settings); Requires `falcon_api.cloud` in CRD spec is defined           |

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

The Falcon Image Analyzer Image is distributed by CrowdStrike through CrowdStrike Falcon registry. Operator supports two modes of deployment:

#### (Option 1) Use CrowdStrike registry directly

Does not require any advanced setup. Users are advised to use the following except in their FalconImageAnalyzer custom resource definition.

```yaml
registry:
  type: crowdstrike
```

The Falcon Image Analyzer product will then be installed directly from CrowdStrike registry. Any new deployment to the cluster may contact CrowdStrike registry for the image download.

#### (Option 2) Let operator mirror Falcon Image Analyzer image to your local registry

Requires advanced setup to grant the operator push access to your local registry. The operator will then mirror the Falcon Image Analyzer image from CrowdStrike registry to your local registry of choice.
Supported registries are: acr, ecr, gcr, and openshift. Each registry type requires advanced setup enable image push.

#### (Option 3) Use a custom Image URI

Image must be available at the specified URI; setting the image attribute will cause registry settings to be ignored. No image mirroring will be leveraged.

Example:
```yaml
image: myprivateregistry.internal.lan/falcon-image-analyzer/falcon-imageanalyzer:1.0.9
```

### Install Steps
To install Falcon Image Analyzer, run the following command to install the FalconImageAnalyzer CR:
```sh
oc create -f https://raw.githubusercontent.com/crowdstrike/falcon-operator/main/config/samples/falcon_v1alpha1_falconimageanalyzer.yaml --edit=true
```

### Uninstall Steps
To uninstall Falcon Image Analyzer simply remove the FalconImageAnalyzer resource. The operator will uninstall the Falcon Image Analyzer from the cluster.

```sh
oc delete falconimageanalyzer --all
```

### Sensor upgrades

To upgrade the sensor version, simply add and/or update the `version` field in the FalconImageAnalyzer resource and apply the change. Alternatively if the `image` field was used instead of using the Falcon API credentials, add and/or update the `image` field in the FalconImageAnalyzer resource and apply the change. The operator will detect the change and perform the upgrade.

### Troubleshooting

- Falcon Operator modifies the FalconImageAnalyzer CR based on what is happening in the cluster. You can get list the CR, Operator Version, and Sensor version by running the following:

  ```sh
  $ oc get falconimageanalyzer
  NAME                    OPERATOR VERSION   FALCON SENSOR
  falcon-image-analyzer   0.8.0              1.0.9
  ```

  This is helpful information to use as a starting point for troubleshooting.
  You can get more insight by viewing the FalconImageAnalyzer CRD in full detail by running the following command:

  ```sh
  oc get falconimageanalyzer -o yaml
  ```

- To review the logs of Falcon Operator:
  ```sh
  oc -n falcon-operator logs -f deploy/falcon-operator-controller-manager -c manager
  ```

- To review the logs of Falcon Image Analyzer service:
  ```sh
  oc logs -n falcon-iar -l "crowdstrike.com/provider=crowdstrike"
  ```

- To review the currently deployed version of the operator:
  ```sh
  oc get falconimageanalyzer -A -o=jsonpath='{.items[].status.version}'
  ```


### Additional Documentation
End-to-end guide(s) to install Falcon-operator together with FalconImageAnalyzer resource.
 - [Deployment Guide for OpenShift](../../README.md)
