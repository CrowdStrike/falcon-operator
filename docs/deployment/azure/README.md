<!--- NOTE: DO NOT EDIT! This file is auto-generated. Please update the source *.tmpl file instead --->
# Deployment Guide for Azure and AKS
This document will guide you through the installation of the Falcon Operator and deployment of the following resources provdied by the Falcon Operator:
- [FalconContainer](../../resources/container/README.md) custom resource to the cluster with Falcon Container image being mirrored from CrowdStrike container registry to ACR (Azure Container Registry).
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

- Create a new FalconContainer resource
  ```sh
  kubectl create -f https://raw.githubusercontent.com/crowdstrike/falcon-operator/main/docs/deployment/azure/falconcontainer.yaml --edit=true
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
