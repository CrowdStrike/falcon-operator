# Deployment Guide for Kubernetes
This document will guide you through the installation of falcon-operator and deployment of either the:
- [FalconContainer](../../resources/container/README.md) custom resource to the cluster with Falcon Container image being pulled from CrowdStrike container registry.
- [FalconNodeSensor](../../resources/node/README.md) custom resource to the cluster.

## Prerequisites

- CrowdStrike CWP subscription
- If your are installing the CrowdStrike Sensor via the Crowdstrike API, you need to create a new CrowdStrike API key pair with the following permissions:
  - Falcon Images Download: Read
  - Sensor Download: Read

## Installing the operator

- Either spin up a Kubernetes cluster or use one that already exists.
- Install the operator
  ```
  kubectl apply -f https://github.com/CrowdStrike/falcon-operator/releases/latest/download/falcon-operator.yaml
  ```

### Deploy the Node Sensor

Once the operator has deployed, you can now deploy the FalconNodeSensor.

- Deploy FalconNodeSensor through the cli using the `kubectl` command:
  ```
  kubectl create -n falcon-operator -f https://raw.githubusercontent.com/CrowdStrike/falcon-operator/main/config/samples/falcon_v1alpha1_falconnodesensor.yaml --edit=true
  ```

### Deploy the sidecar sensor

#### Create the FalconContainer resource

- Create new FalconContainer resource
  ```
  kubectl create -f https://raw.githubusercontent.com/CrowdStrike/falcon-operator/main/docs/deployment/generic/falconcontainer.yaml --edit=true
  ```

## Uninstalling

When uninstalling the operator, it is important to make sure to uninstall the deployed custom resources first *before* you uninstall the operator.
This will insure proper cleanup of the resources.

### Uninstall the Node Sensor

- To uninstall the node sensor, simply remove the FalconNodeSensor resource.
  ```
  kubectl delete falconnodesensor -A --all
  ```

### Uninstall the Sidecar Sensor

- To uninstall Falcon Container, simply remove the FalconContainer resource. The operator will then uninstall the Falcon Container product from the cluster.
  ```
  kubectl delete falconcontainers --all
  ```

### Uninstall the Operator

- To uninstall Falcon Operator, delete the deployment:
  ```
  kubectl delete -f https://github.com/CrowdStrike/falcon-operator/releases/latest/download/falcon-operator.yaml
  ```
