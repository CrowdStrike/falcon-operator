# Installation and Deployment Guide

## Installation and Deployment

The Falcon Operator offers various installation and deployment options for specific Kubernetes distributions. The guides below provide detailed instructions for each case.

> :warning: If none of the guides provide installation for your specific Kubernetes distribution, use the [Deployment Guide for Generic Kubernetes](./deployment/generic/README.md).

### Kubernetes Distribution Installation and Deployment

For an optimal experience, use the following preferred methods when installing for specific Kubernetes Distributions:

- **[Deployment Guide for AKS/ACR](./deployment/azure/README.md)**
- **[Deployment Guide for EKS/ECR](./deployment/eks/README.md)**
- **[Deployment Guide for EKS Fargate](./deployment/eks-fargate/README.md)**
- **[Deployment Guide for GKE/GCR](./deployment/gke/README.md)**
- **[Deployment Guide for OpenShift](./deployment/openshift/README.md)**
- **[Deployment Guide for Generic Kubernetes](./deployment/generic/README.md)**

## Upgrading

Currently, the CrowdStrike Falcon Operator does not support operator upgrades. To upgrade the operator, perform the following steps:

1. Uninstall the deployed custom resources, the operator, and the CRDs (if they still exist).
2. Install the newer operator and re-deploy the custom resources.

## FAQ - Frequently Asked Questions

### What network connections are required for the operator to work properly?

- The operator image is hosted at [quay.io/crowdstrike/falcon-operator](https://quay.io/crowdstrike/falcon-operator). If necessary, the operator image itself can be mirrored to your registry of choice, including internally hosted registries.
- The operator must access your specific Falcon cloud region (`api.crowdstrike.com` or `api.[YOUR CLOUD].crowdstrike.com`).
- Depending on whether the image is mirrored, the operator or your nodes may need access to `registry.crowdstrike.com`.
- If Falcon Cloud is set to autodiscover, the operator may also attempt to reach the Falcon Cloud Region **us-1**.
- If a proxy is configured, please ensure appropriate connections are allowed to Falcon Cloud; otherwise, the operator or custom resource may not deploy correctly.

## Troubleshooting

To review the logs of Falcon Operator:

```shell
kubectl -n falcon-operator logs -f deploy/falcon-operator-controller-manager -c manager
```

### Operator Issues

#### Resources stuck in PodInitializing state indefinitely

If a cluster-wide nodeSelector policy is in place, this must be disabled in the namespaces that the sensors are deployed.

For example, on OpenShift:

```shell
oc annotate ns falcon-operator openshift.io/node-selector=""
```

#### ERROR setup failed to get watch namespace

If the following error shows up in the controller manager logs:

```terminal
1.650281912313243e+09 ERROR setup failed to get watch namespace {"error": "WATCH_NAMESPACE must be set"}
1.6502819123132205e+09 INFO version go {"version": "go1.17.9 linux/amd64"}
1.6502819123131733e+09 INFO version operator {"version": "0.5.0-de97605"}
```

Make sure that the environment variable exists in the controller manager deployment. If it does not exist, add it by running:

```shell
kubectl edit deployment falcon-operator-controller-manager -n falcon-operator
```

and add something similar to the following lines:

```yaml
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

```yaml
        env:
          - name: WATCH_NAMESPACE
            valueFrom:
              fieldRef:
                fieldPath: metadata.namespace
```

The operator is configured to be namespace-scoped and not cluster-scoped which is required for the FalconContainer CR.

This problem can be fixed by running:

```shell
kubectl edit deployment falcon-operator-controller-manager -n falcon-operator
```

and changing `WATCH_NAMESPACE` to the following lines:

```yaml
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
