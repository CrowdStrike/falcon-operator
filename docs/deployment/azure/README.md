# Deployment Guide for Azure and AKS
This document will guide you through the installation of falcon-operator and deployment of either the:
- [FalconContainer](../../resources/container/README.md) custom resource to the cluster with Falcon Container image being mirrored from CrowdStrike container registry to ACR (Azure Container Registry).
- [FalconNodeSensor](../../resources/node/README.md) custom resource to the cluster.

## Prerequisites

- CrowdStrike CWP subscription
- If your are installing the CrowdStrike Sensor via the Crowdstrike API, you need to create a new CrowdStrike API key pair with the following permissions:
  - Falcon Images Download: Read
  - Sensor Download: Read

## Installing the operator

- Either spin up an AKS Kubernetes cluster or use one that already exists.
- Install the operator
  ```sh
  kubectl apply -f https://github.com/CrowdStrike/falcon-operator/releases/latest/download/falcon-operator.yaml
  ```

### Deploy the Node Sensor

Once the operator has deployed, you can now deploy the FalconNodeSensor.

- Deploy FalconNodeSensor through the cli using the `kubectl` command:
  ```sh
  kubectl create -n falcon-operator -f https://raw.githubusercontent.com/CrowdStrike/falcon-operator/main/config/samples/falcon_v1alpha1_falconnodesensor.yaml --edit=true
  ```

### Deploy the sidecar sensor
#### Configure ACR Registry

- Either create or use an existing ACR registry. Make sure to store the ACR registry name in an environment variable.
  ```sh
  ACR_NAME=my-acr-registy-name
  ```

#### Manual installation of ACR push secret

The Image push secret is used by the operator to mirror the Falcon Container sensor image from CrowdStrike registry to your Azure ACR registry. We recommend creating separate service principal just for that task.

- Create kubernetes namespace for falcon-operator

  ```sh
  export FALCON_SYSTEM=falcon-system
  kubectl create ns $FALCON_SYSTEM --dry-run=client -o yaml | kubectl apply -f -
  ```

- Create the service principal in Azure for the CrowdStrike Falcon operator
  ```sh
  # https://docs.microsoft.com/en-us/azure/container-registry/container-registry-auth-service-principal
  SERVICE_PRINCIPAL_NAME=falcon-operator

  ACR_REGISTRY_ID=$(az acr show --name $ACR_NAME --query id --output tsv)
  SP_APP_ID=$(az ad sp list --display-name $SERVICE_PRINCIPAL_NAME --query [].appId --output tsv)
  if ! [ -z "$SP_APP_ID" ]; then
      az ad sp delete --id $SP_APP_ID
  fi

  SP_PASSWD=$(az ad sp create-for-rbac --name $SERVICE_PRINCIPAL_NAME --scopes $ACR_REGISTRY_ID --role acrpush --query password --output tsv)
  SP_APP_ID=$(az ad sp list --display-name $SERVICE_PRINCIPAL_NAME --query [].appId --output tsv)

  # TODO backup docker config
  docker login ... # TODO: script login to your ACR registry
  
  kubectl create secret generic builder --from-file=.dockerconfigjson=$HOME/.docker/config.json --type=kubernetes.io/dockerconfigjson -n $FALCON_SYSTEM

  # TODO restore docker config from the backup
  ```

#### Create the FalconContainer resource

- Create new FalconContainer resource
  ```sh
  kubectl create -f https://raw.githubusercontent.com/CrowdStrike/falcon-operator/main/docs/deployment/azure/falconcontainer.yaml --edit=true
  ```

## Uninstalling

When uninstalling the operator, it is important to make sure to uninstall the deployed custom resources first *before* you uninstall the operator.
This will insure proper cleanup of the resources.

### Uninstall the Node Sensor

- To uninstall the node sensor, simply remove the FalconNodeSensor resource.
  ```sh
  kubectl delete falconnodesensor -A --all
  ```

### Uninstall the Sidecar Sensor

- To uninstall Falcon Container, simply remove the FalconContainer resource. The operator will then uninstall the Falcon Container product from the cluster.
  ```sh
  kubectl delete falconcontainers --all
  ```

### Uninstall the Operator

- To uninstall Falcon Operator, delete the deployment:
  ```sh
  kubectl delete -f https://github.com/CrowdStrike/falcon-operator/releases/latest/download/falcon-operator.yaml
  ```
