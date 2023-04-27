# Deployment Guide for Kubernetes

This document provides a comprehensive guide for installing the Falcon Operator and deploying one of the following custom resources to your cluster:

- [FalconContainer](../../resources/container/README.md)
  > This custom resource pulls the Falcon Container image from the CrowdStrike container registry.
- [FalconNodeSensor](../../resources/node/README.md)
  > This custom resource is deployed directly to the cluster.

## Prerequisites

- CrowdStrike CWP subscription
- CrowdStrike API Key Pair (*if installing the CrowdStrike Sensor via the CrowdStrike API*)

    > If you need help creating a new API key pair, review our docs: [CrowdStrike Falcon](https://falcon.crowdstrike.com/support/api-clients-and-keys).

    Make sure to assign the following permissions to the key pair:
  - Falcon Images Download: **Read**
  - Sensor Download: **Read**

## Installing the Falcon Operator

1. Set up a new Kubernetes cluster or use an existing one.
1. Install the Falcon Operator by running the following command:

    ```shell
    kubectl apply -f https://github.com/CrowdStrike/falcon-operator/releases/latest/download/falcon-operator.yaml
    ```

### Deploying the Falcon Node Sensor

After the Falcon Operator is deployed, run the following command to deploy the Falcon Node Sensor:

```shell
kubectl create -n falcon-operator -f https://raw.githubusercontent.com/CrowdStrike/falcon-operator/main/config/samples/falcon_v1alpha1_falconnodesensor.yaml --edit=true
```

### Deploying the Falcon Container Sidecar Sensor

To deploy the Falcon Container Sidecar Sensor, run the following command:

```shell
kubectl create -f https://raw.githubusercontent.com/CrowdStrike/falcon-operator/main/docs/deployment/generic/falconcontainer.yaml --edit=true
```

## Uninstalling the Falcon Operator and Resources

> :exclamation: It is essential to uninstall the deployed custom resources before uninstalling the Falcon Operator to ensure proper cleanup.

### Uninstalling the Falcon Node Sensor

Remove the FalconNodeSensor resource by running:

```shell
kubectl delete falconnodesensor -A --all
```

### Uninstalling the Falcon Container Sidecar Sensor

Remove the FalconContainer resource. The operator will then uninstall the Falcon Container product from the cluster:

```shell
kubectl delete falconcontainers --all
```

### Uninstalling the Falcon Operator

Delete the Falcon Operator deployment by running:

```shell
kubectl delete -f https://github.com/CrowdStrike/falcon-operator/releases/latest/download/falcon-operator.yaml
```
