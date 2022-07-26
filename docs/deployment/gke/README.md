# Deployment Guide for GKE
This document will guide you through the installation of falcon-operator and deployment of [FalconContainer](../../container) custom resource to the cluster with Falcon Container image being mirrored from CrowdStrike container registry to GCR (Google Container Registry). New GCP service account for pushing to GCR registry will be created.

## Pre-requisites

 - Have CrowdStrike CWP subscription with Falcon Container enabled
 - Have Container Administrator access to GCP and at least one GKE cluster deployed
 - Create new CrowdStrike API key pair with the following permissions
    * Falcon Images Download: Read
    * Sensor Download: Read

## Installation Steps

 - Open GCP Cloud Shell: https://shell.cloud.google.com/?hl=en_US&fromcloudshell=true&show=terminal
 - Ensure the Cloud Shell is running in context of GCP project you want to use
   ```
   gcloud config get-value core/project
   ```
 - In case you have multiple GKE clusters in the project, You need to select the desired one to install the operator in
   ```
   gcloud container clusters get-credentials DESIRED_CLUSTER --zone DESIRED_LOCATION
   ```
 - Install the operator & operator-sdk & deploy Falcon Container Sensor
   ```
   bash -c 'source <(curl -s https://raw.githubusercontent.com/CrowdStrike/falcon-operator/main/docs/deployment/gke/run)'
   ```

  Note :
   - By default, Falcon Container sensor injector is configured to monitor all namespaces. For GKE cluster/Node upgrade, explicitly label the kube-public and kube-system namespace to not be monitored by Crowdstrike. Also falcon-operator and falcon-system namespaces should be labeled to disabled.

    kubectl label namespace falcon-operator sensor.falcon-system.crowdstrike.com/injection=disabled
    kubectl label namespace falcon-system sensor.falcon-system.crowdstrike.com/injection=disabled
    kubectl label namespace kube-system sensor.falcon-system.crowdstrike.com/injection=disabled
    kubectl label namespace kube-public sensor.falcon-system.crowdstrike.com/injection=disabled

    This will ensure that, any pod related to k8 control plane and Falcon are not forwarded to the injector

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
   kubectl delete -f https://raw.githubusercontent.com/CrowdStrike/falcon-operator/main/deploy/falcon-operator.yaml
   ```

## Manual installation of GCR push secret

If you don't want to use the installation [script](run) mentioned above you may need to create image push secret manually.

Image push secret is used by the operator to mirror Falcon Container image from CrowdStrike registry to your GCR.

 - Set environment variable to refer to your GCP project
   ```
   GCP_PROJECT_ID=$(gcloud config get-value core/project)
   ```
 - Create new GCP service account
   ```
   gcloud iam service-accounts create falcon-operator
   ```
 - Grant image push access to the newly created service account
   ```
   gcloud projects add-iam-policy-binding $GCP_PROJECT_ID \
       --member serviceAccount:falcon-operator@$GCP_PROJECT_ID.iam.gserviceaccount.com \
       --role roles/storage.admin
   ```
 - Create new private key for the newly create service account
   ```
   gcloud iam service-accounts keys create \
       --iam-account "falcon-operator@$GCP_PROJECT_ID.iam.gserviceaccount.com" \
       .dockerconfigjson
   ```
 - Store the newly created private key for image push in the kubernetes
   ```
   kubectl create secret docker-registry -n falcon-system-configure builder --from-file .dockerconfigjson
   ```

## Granting GCP Workload Identity to Falcon Container Injector

Falcon Container Injector may need [GCP Workload Identity](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity)
to read GCR or Artifact Registry. In many cases this GCP Workload Identity is assigned or inherited automatically. However, if you
are seeing errors similar to the following you may need to follow this guide to assign the identity manually.

```
time="2022-01-14T13:05:11Z" level=error msg="Failed to handle webhook request" error="Failed to retrieve image details for \"gcr.io/\" in container in pod: Failed to get the image config/digest for \"gcr.io/" on \"eu.gcr.io\": Error reading manifest latest in gcr.io/: unauthorized: You don't have the needed permissions to perform this operation, and you may have invalid credentials. To authenticate your request, follow the steps in: https://cloud.google.com/container-registry/docs/advanced-authentication"
```

### Assigning GCP Workload Identity to Falcon Container Injector

Conceptually, the following tasks need to be done in order to enable GCR pull from the injector

 - Create GCP Service Account
 - Grant GCR permissions to the newly created Service Account
 - Allow Falcon Container to use the newly created Service Account
 - Put GCP Service Account handle into your Falcon Container resource for re-deployments

The following step-by-step guide uses `gcloud`, and `kubectl` command-line tools to achieve that.

### Step-by-step guide

 - Set up your shell environment variables
   ```
   GCP_SERVICE_ACCOUNT=falcon-container-injector

   GCP_PROJECT_ID=$(gcloud config get-value core/project)
   ```

 - Create new GCP Service Account
   ```
   gcloud iam service-accounts create $GCP_SERVICE_ACCOUNT
   ```

 - Grant GCR permissions to the newly created Service Account
   ```
   gcloud projects add-iam-policy-binding $PROJECT_ID \
       --member "serviceAccount:$GCP_SERVICE_ACCOUNT@$GCP_PROJECT_ID.iam.gserviceaccount.com" \
       --role roles/containerregistry.ServiceAgent
   ```

 - Allow Falcon Injector to use the newly created GCP Service Account
   ```
   gcloud iam service-accounts add-iam-policy-binding \
       $GCP_SERVICE_ACCOUNT@$GCP_PROJECT_ID.iam.gserviceaccount.com \
       --role roles/iam.workloadIdentityUser \
       --member "serviceAccount:$GCP_PROJECT_ID.svc.id.goog[falcon-system/default]"
   ```

- Re-deploy (delete & create) FalconContainer with the above Service Account added to the spec:

  Delete FalconContainer
  ```
  kubectl delete falconcontainers --all
  ```

  Add Newly created Service Account to your FalconContainer yaml file:
  ```
  spec:
    injector:
      sa_annotations:
        iam.gke.io/gcp-service-account: $GCP_SERVICE_ACCOUNT@$GCP_PROJECT_ID.iam.gserviceaccount.com
  ```

  (don't forget to replace the service account name template with actual name)
  ```
  echo "$GCP_SERVICE_ACCOUNT@$GCP_PROJECT_ID.iam.gserviceaccount.com"
  ```
