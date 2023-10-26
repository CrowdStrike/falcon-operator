<!--- NOTE: DO NOT EDIT! This file is auto-generated. Please update the source *.tmpl file instead --->
# Deployment Guide for Kubernetes
This document will guide you through the installation of the Falcon Operator and deployment of the following resources provdied by the Falcon Operator:
- [FalconContainer](../../resources/container/README.md) custom resource to the cluster with Falcon Container image being mirrored from CrowdStrike container registry to .
- [FalconNodeSensor](../../resources/node/README.md) custom resource to the cluster.

## Prerequisites

- CrowdStrike CWP subscription
- CrowdStrike API Key Pair (*if installing the CrowdStrike Sensor via the CrowdStrike API*)

    > If you need help creating a new API key pair, review our docs: [CrowdStrike Falcon](https://falcon.crowdstrike.com/support/api-clients-and-keys).

  Make sure to assign the following permissions to the key pair:
  - Falcon Images Download: **Read**
  - Sensor Download: **Read**

## Installing the Falcon Operator

- Set up a new Kubernetes cluster or use an existing one.

- Install the Falcon Operator by running the following command:
  ```sh
  kubectl apply -f https://github.com/crowdstrike/falcon-operator/releases/latest/download/falcon-operator.yaml
  ```

### Deploying the Falcon Node Sensor

After the Falcon Operator has deployed, you can now deploy the Falcon Node Sensor:

- Deploy FalconNodeSensor through the cli using the `kubectl` command:
  ```sh
  kubectl create -n falcon-operator -f https://raw.githubusercontent.com/crowdstrike/falcon-operator/main/config/samples/falcon_v1alpha1_falconnodesensor.yaml --edit=true
  ```

### Deploying the Falcon Container Sidecar Sensor

#### Create the FalconContainer resource

- Create a new FalconContainer resource
  ```sh
  kubectl create -f https://raw.githubusercontent.com/crowdstrike/falcon-operator/main/docs/deployment/generic/falconcontainer.yaml --edit=true
  ```



## Uninstalling

> :exclamation: It is essential to uninstall ALL of the deployed custom resources before uninstalling the Falcon Operator to ensure proper cleanup.

### Uninstalling the Falcon Node Sensor

Remove the FalconNodeSensor resource by running:

```sh
kubectl delete falconnodesensor -A --all
```

### Uninstalling the Falcon Container Sidecar Sensor

Remove the FalconContainer resource. The operator will then uninstall the Falcon Container Sidecar Sensor from the cluster:

```sh
kubectl delete falconcontainers --all
```

### Uninstalling the Falcon Operator

Delete the Falcon Operator deployment by running:

```sh
kubectl delete -f https://github.com/crowdstrike/falcon-operator/releases/latest/download/falcon-operator.yaml
```
