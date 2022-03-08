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
| falcon_api.client_id                | CrowdStrike API Client ID                                                                                                                 |
| falcon_api.client_secret            | CrowdStrike API Client Secret                                                                                                             |
| falcon_api.client_region            | CrowdStrike cloud region (allowed values: autodiscover, us-1, us-2, eu-1, us-gov-1)                                                       |
| falcon.cid                          | (optional) CrowdStrike Falcon CID override                                                                                                |
| node.image_override                 | (optional) Location of the Falcon Sensor Image. Specify only when you mirror the original image to your own image repository              |

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

To review the logs of Falcon Operator:
```
kubectl -n falcon-operator logs -f deploy/falcon-operator-controller-manager -c manager
```
