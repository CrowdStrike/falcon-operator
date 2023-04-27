# Installation and Deployment Guide

## Installation and Deployment

There are several ways to install and deploy the Falcon Operator. The guides below will walk you through installing for specific Kubernetes distributions.
> :warning: If none of the guides provide installation for your specific Kubernetes distribution, use the [Deployment Guide for Generic Kubernetes](./deployment/generic/README.md).

### Kubernetes Distribution Installation and Deployment

The following are the preferred methods for installing for specific Kubernetes Distributions. Please use these over other methods of installation.

 - **[Deployment Guide for AKS/ACR](./deployment/azure/README.md)**
 - **[Deployment Guide for EKS/ECR](./deployment/eks/README.md)**
 - **[Deployment Guide for EKS Fargate](./deployment/eks-fargate/README.md)**
 - **[Deployment Guide for GKE/GCR](./deployment/gke/README.md)**
 - **[Deployment Guide for OpenShift](./deployment/openshift/README.md)**
 - **[Deployment Guide for Generic Kubernetes](./deployment/generic/README.md)**

## Upgrading

The CrowdStrike Falcon Operator does not currently support upgrading the operator. To upgrade the operator, you must:

1. Uninstall the deployed custom resources, the operator, and the CRDs if they still exist.
2. Install the newer operator and re-deploy the custom resources.

## FAQ - Frequently Asked Questions

### What network connections are required for the operator to work properly?

 - The operator image is available at [quay.io/crowdstrike/falcon-operator](https://quay.io/crowdstrike/falcon-operator). If necessary, the operator image itself can be mirrored to your registry of choice, including internally hosted registries.
 - The operator will need to be able to reach your particular Falcon cloud region (api.crowdstrike.com or api.[YOUR CLOUD].crowdstrike.com).
 - The operator OR your nodes may attempt to reach to `registry.crowdstrike.com` depending on whether the image is being mirrored or not.
 - If Falcon Cloud is set to autodiscover, the operator may also attempt to reach the Falcon Cloud Region **us-1**.
 - If a proxy is configured, please make sure that all the appropriate connections are allowed to the Falcon Cloud, or the operator or custom resource may fail to deploy correctly.

## Troubleshooting

To review the logs of Falcon Operator:
```
kubectl -n falcon-operator logs -f deploy/falcon-operator-controller-manager -c manager
```

### Operator Issues

#### Resources stuck in PodInitializing state indefinitely

If a cluster-wide nodeSelector policy is in place, this must be disabled in the namespaces that the sensors are deployed.

For example, on OpenShift:
```
oc annotate ns falcon-operator openshift.io/node-selector=""
```

#### ERROR setup failed to get watch namespace

If the following error shows up in the controller manager logs:
```
1.650281912313243e+09 ERROR setup failed to get watch namespace {"error": "WATCH_NAMESPACE must be set"}
1.6502819123132205e+09 INFO version go {"version": "go1.17.9 linux/amd64"}
1.6502819123131733e+09 INFO version operator {"version": "0.5.0-de97605"}
```
Make sure that the environment variable exists in the controller manager deployment. If it does not exist, add it by running:
```
kubectl edit deployment falcon-operator-controller-manager -n falcon-operator
```
and add something similar to the following lines:
```
        env:
          - name: WATCH_NAMESPACE
            value: ''
          - name: POD_NAME
            valueFrom:
              fieldRef:
                fieldPath: metadata.name
          - name: OPERATOR_NAME
            value: "falcon-operator"
```

#### FalconContainer is stuck in the CONFIGURING Phase

Make sure that the `WATCH_NAMESPACE` variable is correctly configured to be cluster-scoped and not namespace-scoped. If the
controller manager's deployment has the following configuration:
```
        env:
          - name: WATCH_NAMESPACE
            valueFrom:
              fieldRef:
                fieldPath: metadata.namespace
```
the operator is configured to be namespace-scoped and not cluster-scoped which is required for the FalconContainer CR.
This problem can be fixed by running:
```
kubectl edit deployment falcon-operator-controller-manager -n falcon-operator
```
and changing `WATCH_NAMESPACE` to the following lines:
```
        env:
          - name: WATCH_NAMESPACE
            value: ''
          - name: POD_NAME
            valueFrom:
              fieldRef:
                fieldPath: metadata.name
          - name: OPERATOR_NAME
            value: "falcon-operator"
```
Once a new version of the controller manager has deployed, you may have to delete and recreate the FalconContainer CR.
