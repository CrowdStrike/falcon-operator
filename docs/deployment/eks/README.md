# Deployment Guide for EKS and ECR
This document will guide you through the installation of falcon-operator and deployment of either the:
- [FalconContainer](../../cluster_resources/container/README.md) custom resource to the cluster with Falcon Container image being mirrored from CrowdStrike container registry to ECR (Elastic Container Registry). A new AWS IAM Policy will be created to allow the opeator to push to ECR registry.
- [FalconNodeSensor](../../cluster_resources/node/README.md) custom resource to the cluster.

## Prerequisites

- CrowdStrike CWP subscription
- If your are installing the CrowdStrike Sensor via the Crowdstrike API, you need to create a new CrowdStrike API key pair with the following permissions:
  - Falcon Images Download: Read
  - Sensor Download: Read

## Installing the operator

- Either spin up an EKS Kubernetes cluster or use one that already exists. The EKS cluster that runs Falcon Operator needs to have the [IAM OIDC provider](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html) installed. The IAM OIDC provider associates AWS IAM roles with EKS workloads. Please review [AWS documentation](https://docs.aws.amazon.com/eks/latest/userguide/enable-iam-roles-for-service-accounts.html) to understand how the IAM OIDC provider works before proceeding.

 - Provide the following AWS settings as environment variables:
  ```
  export AWS_REGION=<my_aws_region>
  export EKS_CLUSTER_NAME=<my_cluster_name>
  ```

 - Install IAM OIDC on the cluster if it is not already installed:
  ```
  eksctl utils associate-iam-oidc-provider --region "$AWS_REGION" --cluster "$EKS_CLUSTER_NAME" --approve
  ```

- Install the operator
  ```
  kubectl apply -f https://raw.githubusercontent.com/CrowdStrike/falcon-operator/main/deploy/falcon-operator.yaml
  ```

### Deploy the Node Sensor

Once the operator has deployed, you can now deploy the FalconNodeSensor.

- Deploy FalconNodeSensor through the cli using the `kubectl` command:
  ```
  kubectl create -n falcon-operator -f https://raw.githubusercontent.com/CrowdStrike/falcon-operator/main/config/samples/falcon_v1alpha1_falconnodesensor.yaml --edit=true
  ```

### Deploying the Node Sensor to a custom Namespace

If desired, the FalconNodeSensor can be deployed to a namespace of your choosing instead of deploying to the operator namespace.
To deploy to a custom namespace (replacing `falcon-system` as desired):

- Create a new project
  ```
  kubectl create namespace falcon-system
  ```

- Create the service account in the new namespace
  ```
  kubectl create sa falcon-operator-node-sensor -n falcon-system
  ```

- Deploy FalconNodeSensor to the custom namespace:
  ```
  kubectl create -n falcon-system -f https://raw.githubusercontent.com/CrowdStrike/falcon-operator/main/docs/config/samples/falcon_v1alpha1_falconnodesensor.yaml --edit=true
  ```

### Deploy the sidecar sensor
#### Create the FalconContainer resource

- Create new FalconContainer resource
  ```
  kubectl create -f https://raw.githubusercontent.com/CrowdStrike/falcon-operator/main/docs/deployment/eks/falconcontainer.yaml --edit=true
  ```

#### Complete install using AWS Cloud Shell

 - Open AWS Cloud Shell: https://console.aws.amazon.com/cloudshell/home

 - Install the operator & deploy Falcon Container Sensor
   ```
   bash -c 'source <(curl -s https://raw.githubusercontent.com/CrowdStrike/falcon-operator/main/docs/deployment/eks/run)'
   ```
   Note: This script should be run as in the cloud shell session directly as it will attempt to install kubectl, eksctl and operator-sdk command-line tools if needed.

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
  kubectl delete -f https://raw.githubusercontent.com/CrowdStrike/falcon-operator/main/deploy/falcon-operator.yaml
  ```
