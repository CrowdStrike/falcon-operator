# Falcon Node Sensor

## About Falcon Cloud Workload Protection

Learn more at [product page](https://www.crowdstrike.co.uk/cloud-security-products/falcon-cloud-workload-protection/) and [Linux sensor blog](https://www.crowdstrike.com/blog/tech-center/linux-protection/).

## About FalconNodeSensor Custom Resource
Falcon Operator introduces FalconNodeSensor Custom Resource to the cluster. The resource is meant to be singleton and it will install, configure and uninstall Falcon Linux Sensor on the cluster nodes.

Example:
```
apiVersion: falcon.crowdstrike.com/v1alpha1
kind: FalconNodeSensor
metadata:
  name: falcon-node-sensor
spec:
  # Add fields here
  node:
    terminationGracePeriod: 30
  falcon:
    apd: null
    aph: null
    app: null
    billing: null
    cid: null
    feature: null
    message_log: null
    provisioning_token: null
    tags: null
    trace: none
```

### FalconNodeSensor Reference Manual

| Spec                                | Description                                                                                                                               |
| :---------------------------------- | :---------------------------------------------------------------------------------------------------------------------------------------- |
| falcon.cid                          | CrowdStrike Falcon CID                                                                                                                    |
| node.image                          | Location of the CrowdStrike Daemonset image                                                                                               |

### Install Steps
To install Falcon Node Sensor (assuming Falcon Operator is installed):
```
kubectl create -f https://raw.githubusercontent.com/CrowdStrike/falcon-operator/main/config/samples/falcon_v1alpha1_falconnodesensor.yaml --edit=true
```

### Uninstall Steps
To uninstall Falcon Node Sensor simply remove the FalconNodeSensor resource. The operator will uninstall the Falcon Sensor from the cluster.

```
kubectl delete falconnodesensors.falcon.crowdstrike.com --all -A
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
