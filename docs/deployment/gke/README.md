# Deployment Guide for GKE
This document will guide you through the installation of falcon-operator and deployment of [FalconContainer](../../container) custom resource to the cluster with Falcon Container image being mirrored from CrowdStrike container registry to GCR (Google Container Registry). New GCP service account for pushing to GCR registry will be created.

## Pre-requisites

 - Have CrowdStrike CWP subscription with Falcon Container enabled
 - Have Container Administrator access to GCP and at least one GKE cluster deployed
 - Create new CrowdStrike API key pair with permission to download the sensor (no other permission shall be required)

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
