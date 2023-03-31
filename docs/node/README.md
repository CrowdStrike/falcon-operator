# Falcon Node Sensor

## About Falcon Cloud Workload Protection

Learn more at [product page](https://www.crowdstrike.co.uk/cloud-security-products/falcon-cloud-workload-protection/) and [Linux sensor blog](https://www.crowdstrike.com/blog/tech-center/linux-protection/).

## About FalconNodeSensor Custom Resource
Falcon Operator introduces FalconNodeSensor Custom Resource to the cluster. The resource is meant to be singleton and it will install, configure and uninstall Falcon Linux Sensor on the cluster nodes.

To start the FalconNodeSensor installation please push the following FalconNodeSensor resource to your cluster. You will need to provide CrowdStrike API Keys and CrowdStrike cloud region for the installation. It is recommended to establish new API credentials for the installation at https://falcon.crowdstrike.com/support/api-clients-and-keys, required permissions are:
 * Falcon Images Download: Read
 * Sensor Download: Read

Example:
```
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

### FalconNodeSensor Reference Manual

| Spec                                | Description                                                                                                                               |
| :---------------------------------- | :---------------------------------------------------------------------------------------------------------------------------------------- |
| falcon_api.client_id                | (optional) CrowdStrike API Client ID                                                                                                      |
| falcon_api.client_secret            | (optional) CrowdStrike API Client Secret                                                                                                  |
| falcon_api.cloud_region             | (optional) CrowdStrike cloud region (allowed values: autodiscover, us-1, us-2, eu-1, us-gov-1)                                            |
| falcon.cid                          | (optional) CrowdStrike Falcon CID override                                                                                                |
| falcon.apd                          | (optional) Configure Falcon Sensor to leverage a proxy host                                                                               |
| falcon.aph                          | (optional) Configure the host Falcon Sensor should leverage for proxying                                                                  |
| falcon.app                          | (optional) Configure the port Falcon Sensor should leverage for proxying                                                                  |
| falcon.billing                      | (optional) Configure Pay-as-You-Go (metered) billing rather than default billing                                                          |
| falcon.provisioning_token           | (optional) Configure a Provisioning Token for CIDs with restricted AID provisioning enabled                                               |
| falcon.tags                         | (optional) Configure Falcon Sensor Grouping Tags; comma-delimited                                                                         |
| falcon.trace                        | (optional) Configure Falcon Sensor Trace Logging Level (none, err, warn, info, debug)                                                     |
| node.nodeAffinity                   | (optional) See https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/ for examples on configuring nodeAffinity          |
| node.image_override                 | (optional) Location of the Falcon Sensor Image. Specify only when you mirror the original image to your own image repository              |
| node.imagePullSecrets               | (optional) list of references to secrets in the falcon-system namespace to use for pulling image from image_override location.            |
| node.backend                        | (optional) Configure the backend mode for Falcon Sensor (allowed values: kernel, bpf)                                                     |
| node.disableCleanup                 | (optional) Cleans up `/opt/CrowdStrike` on the nodes by deleting the files and directory.                                                 |
| node.version                        | (optional) Enforce particular Falcon Sensor version to be installed (example: "6.35", "6.35.0-13207")                                     |

All arguments are optional, but successful deployment requires either falcon_id and falcon_secret **or** cid and image_override. In a former case container image and cid will be fetched from CrowdStrike Falcon API. While in the latter case the CID and image location is explicitly specified by the user.

### Install Steps
To install Falcon Node Sensor (assuming Falcon Operator is installed):
```
kubectl create -f https://raw.githubusercontent.com/CrowdStrike/falcon-operator/main/config/samples/falcon_v1alpha1_falconnodesensor.yaml --edit=true
```

### Uninstall Steps
To uninstall Falcon Node Sensor simply remove the FalconNodeSensor resource. The operator will uninstall the Falcon Sensor from the cluster.

```
kubectl delete falconnodesensors.falcon.crowdstrike.com --all
```

### Troubleshooting

To see the FalconNodeSensor resource on the cluster
```
kubectl get falconnodesensors.falcon.crowdstrike.com -A
```

To verify the existence of the daemonset object
```
kubectl get daemonsets.apps -n falcon-system
```

To verify the existence of the sensor objects
```
kubectl get pods -n falcon-system
```

To review the logs of Falcon Operator:
```
kubectl -n falcon-operator logs -f deploy/falcon-operator-controller-manager -c manager
```

To review the currently deployed version of the operator
```
kubectl get deployments -n falcon-operator falcon-operator-controller-manager -o=jsonpath='{.spec.template.spec.containers[].image}'
```
