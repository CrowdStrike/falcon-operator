<!--- NOTE: DO NOT EDIT! This file is auto-generated. Please update the source *.tmpl file instead --->
# Deployment Guide for EKS and ECR
This document will guide you through the installation of the Falcon Operator and deployment of the following custom resources provided by the Falcon Operator:
- [FalconAdmission](../../resources/admission/README.md) with the Falcon Admission Controller image being mirrored from CrowdStrike container registry to ECR (Elastic Container Registry). A new AWS IAM Policy will be created to allow the operator to push to ECR registry.
- [FalconContainer](../../resources/container/README.md) with the Falcon Container image being mirrored from CrowdStrike container registry to ECR (Elastic Container Registry). A new AWS IAM Policy will be created to allow the operator to push to ECR registry.
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

- Set up a new Kubernetes cluster or use an existing one. The EKS cluster that runs Falcon Operator needs to have the [IAM OIDC provider](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html) installed. The IAM OIDC provider associates AWS IAM roles with EKS workloads.
Please review [AWS documentation](https://docs.aws.amazon.com/eks/latest/userguide/enable-iam-roles-for-service-accounts.html) to understand how the IAM OIDC provider works before proceeding.

 - Provide the following AWS settings as environment variables:
  ```sh
  export AWS_REGION=<my_aws_region>
  export EKS_CLUSTER_NAME=<my_cluster_name>
  ```

 - Install IAM OIDC on the cluster if it is not already installed:
  ```sh
  eksctl utils associate-iam-oidc-provider --region "$AWS_REGION" --cluster "$EKS_CLUSTER_NAME" --approve
  ```

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
  kubectl create -f https://raw.githubusercontent.com/crowdstrike/falcon-operator/main/docs/deployment/eks/falconcontainer.yaml --edit=true
  ```


#### Complete install using AWS Cloud Shell

 - Open AWS Cloud Shell: https://console.aws.amazon.com/cloudshell/home

 - Install the operator & deploy Falcon Container Sensor
   ```sh
   bash -c 'source <(curl -s https://raw.githubusercontent.com/crowdstrike/falcon-operator/main/docs/deployment/eks/run)'
   ```
   > [!NOTE]
   > This script should be run as in the cloud shell session directly as some command line tools may be installed in the process.

</details>

### Deploying the Falcon Admission Controller

<details>
  <summary>Click to expand</summary>

- Create a new FalconAdmission resource
  ```sh
  kubectl create -f https://raw.githubusercontent.com/crowdstrike/falcon-operator/main/docs/deployment/eks/falconadmission.yaml --edit=true
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
