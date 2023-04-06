# Deployment Guide for Azure/AKS
This document will guide you through the installation of falcon-operator and deployment of [FalconContainer](../../container) custom resource to the cluster with Falcon Container image being mirrored from CrowdStrike container registry to ACR (Azure Container Registry).

## Pre-requisites

 - Have CrowdStrike CWP subscription with Falcon Container enabled


## Installation Steps

 - Spin up a Kubernetes cluster (or use existing)

 - Install the operator

 - Create ACR registry (or use existing) and store the name to environment variable
   ```
   ACR_NAME=my-acr-registy-name
   ```

 - Install ACR push secret

   Please refer to the steps at the bottom of this page.


 - Create new FalconContainer resource
   ```
   kubectl create -f https://raw.githubusercontent.com/CrowdStrike/falcon-operator/f5dbd8f7e37256b52b6db03a163102c333c6051f/docs/deployment/azure/falconcontainer.yaml --edit=true
   ```

## Uninstall Steps

 - To uninstall Falcon Container simply remove FalconContainer resource. The operator will uninstall Falcon Container product from the cluster.
   ```
   kubectl delete falconcontainers.falcon.crowdstrike.com default
   ```
 - To uninstall Falcon Operator that was installed using Operator Lifecycle manager
   ```
   operator-sdk cleanup falcon-operator --namespace falcon-operator
   ```
 - To uninstall Falcon Operator that was installed without Operator Lifecycle manager
   ```
   kubectl delete -f https://github.com/CrowdStrike/falcon-operator/releases/latest/download/falcon-operator.yaml
   ```

## Manual installation of ACR push secret

Image push secret is used by the operator to mirror Falcon Container image from CrowdStrike registry to your ACR. We recommend creating separate service principal just for that task.

 - Create kubernetes namespace for falcon-operator

   ```
   export FALCON_SYSTEM=falcon-system
   kubectl create ns $FALCON_SYSTEM --dry-run=client -o yaml | kubectl apply -f -
   ```

 - Create service principal in Azure for falcon-operator
   ```
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
