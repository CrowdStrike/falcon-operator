<!--- NOTE: DO NOT EDIT! This file is auto-generated. Please update the source *.tmpl file instead --->
# Deployment Guide for Azure and AKS
This document will guide you through the installation of the Falcon Operator and deployment of the following custom resources provided by the Falcon Operator:
- [FalconAdmission](../../resources/admission/README.md) with the Falcon Admission Controller image being mirrored from CrowdStrike container registry to ACR (Azure Container Registry).
- [FalconContainer](../../resources/container/README.md) with the Falcon Container image being mirrored from CrowdStrike container registry to ACR (Azure Container Registry).
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
#### Configure ACR Registry

- Either create or use an existing ACR registry. Make sure to store the ACR registry name in an environment variable.
  ```sh
  ACR_NAME=my-acr-registry-name
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



</details>

### Deploying the Falcon Admission Controller

<details>
  <summary>Click to expand</summary>

- Create a new FalconAdmission resource
  ```sh
  kubectl create -f https://raw.githubusercontent.com/crowdstrike/falcon-operator/main/docs/deployment/azure/falconadmission.yaml --edit=true
  ```

- For Azure AKS workloads, you must disable the Azure Admissions Enforcer for the Falcon Admission Controller.
  Add the following annotation to the validating webhook configuration to disable the Azure Admissions Enforcer for the Falcon Admission Controller:
  ```sh
  kubectl annotate validatingwebhookconfiguration validating.admission.falcon.crowdstrike.com admissions.enforcer/disabled=true
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
