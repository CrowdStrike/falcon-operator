<!--- NOTE: DO NOT EDIT! This file is auto-generated. Please update the source *.tmpl file instead --->
# Deployment Guide for Kubernetes
This document will guide you through the installation of the Falcon Operator and deployment of the following custom resources provided by the Falcon Operator:
- [FalconAdmission](../../resources/admission/README.md) with the Falcon Admission Controller image being mirrored from CrowdStrike container registry to .
- [FalconContainer](../../resources/container/README.md) with the Falcon Container image being mirrored from CrowdStrike container registry to .
- [FalconImageAnalyzer](../../resources/imageanalyzer/README.md) with the Falcon Image Analyzer image being pull from the CrowdStrike container registry.
- [FalconNodeSensor](../../resources/node/README.md) custom resource to the cluster.

## Prerequisites

> [!IMPORTANT]
> - The correct CrowdStrike Cloud (not Endpoint) subscription
> - CrowdStrike API Key Pair (*if installing the CrowdStrike Sensor via the CrowdStrike API*)
>
>    > If you need help creating a new API key pair, review our docs: [CrowdStrike Falcon](https://falcon.crowdstrike.com/support/api-clients-and-keys).
>
>  Make sure to assign the following permissions to the key pair:
>  - Falcon Images Download: **Read**
>  - Sensor Download: **Read**

## Installing the Falcon Operator

<details>
  <summary>Click to expand</summary>

- Set up a new Kubernetes cluster or use an existing one.

- Install the Falcon Operator by running the following command:
  ```sh
  kubectl apply -f https://github.com/crowdstrike/falcon-operator/releases/latest/download/falcon-operator.yaml
  ```

</details>

### Deploying the Falcon Node Sensor

<details>
  <summary>Click to expand</summary>

After the Falcon Operator has deployed, you can now deploy the Falcon Node Sensor:

- Deploy FalconNodeSensor through the cli using the `kubectl` command:
  ```sh
  kubectl create -n falcon-operator -f https://raw.githubusercontent.com/crowdstrike/falcon-operator/main/config/samples/falcon_v1alpha1_falconnodesensor.yaml --edit=true
  ```
</details>

### Deploying the Falcon Container Sidecar Sensor

<details>
  <summary>Click to expand</summary>

#### Create the FalconContainer resource

- Create a new FalconContainer resource
  ```sh
  kubectl create -f https://raw.githubusercontent.com/crowdstrike/falcon-operator/main/docs/deployment/generic/falconcontainer.yaml --edit=true
  ```



</details>

### Deploying the Falcon Admission Controller

<details>
  <summary>Click to expand</summary>

- Create a new FalconAdmission resource
  ```sh
  kubectl create -f https://raw.githubusercontent.com/crowdstrike/falcon-operator/main/docs/deployment/generic/falconadmission.yaml --edit=true
  ```

</details>

### Deploying the Falcon Image Analyzer

<details>
  <summary>Click to expand</summary>

After the Falcon Operator has deployed, you can now deploy the Image Analyzer:

- Deploy FalconImageAnalyzer through the cli using the `kubectl` command:
  ```sh
  kubectl create -n falcon-operator -f https://raw.githubusercontent.com/crowdstrike/falcon-operator/main/config/samples/falcon_v1alpha1_falconimageanalyzer.yaml --edit=true
  ```

</details>

## Upgrading

<details>
  <summary>Click to expand</summary>

To upgrade, run the following command:

```sh
kubectl apply -f https://github.com/crowdstrike/falcon-operator/releases/latest/download/falcon-operator.yaml
```

If you want to upgrade to a specific version, replace `latest` with the desired version number in the URL:

```sh
VERSION=1.2.3
kubectl apply -f https://github.com/CrowdStrike/falcon-operator/releases/download/${VERSION}/falcon-operator.yaml
```

</details>

## Uninstalling

> [!WARNING]
> It is essential to uninstall ALL of the deployed custom resources before uninstalling the Falcon Operator to ensure proper cleanup.

### Uninstalling the Falcon Node Sensor

<details>
  <summary>Click to expand</summary>

Remove the FalconNodeSensor resource by running:

```sh
kubectl delete falconnodesensor -A --all
```

</details>

### Uninstalling the Falcon Container Sidecar Sensor

<details>
  <summary>Click to expand</summary>

Remove the FalconContainer resource. The operator will then uninstall the Falcon Container Sidecar Sensor from the cluster:

```sh
kubectl delete falconcontainers --all
```

</details>

### Uninstalling the Falcon Admission Controller

<details>
  <summary>Click to expand</summary>

Remove the FalconAdmission resource. The operator will then uninstall the Falcon Admission Controller from the cluster:

```sh
kubectl delete falconadmission --all
```

</details>

### Uninstalling the Falcon Image Analyzer

<details>
  <summary>Click to expand</summary>

Remove the FalconImageAnalyzer resource. The operator will then uninstall the Falcon Image Analyzer from the cluster:

```sh
kubectl delete falconimageanalyzer --all
```

</details>

### Uninstalling the Falcon Operator

<details>
  <summary>Click to expand</summary>

Delete the Falcon Operator deployment by running:

```sh
kubectl delete -f https://github.com/crowdstrike/falcon-operator/releases/latest/download/falcon-operator.yaml
```

</details>
