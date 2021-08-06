# Deployment Guide for GKE

## Pre-requisites

 - Have CrowdStrike CWP subscription with Falcon Container enabled (have your CrowdStrike CID ready)
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
   kubectl delete falconcontainers.falcon.crowdstrike.com  -n falcon-system-configure default
   ```
 - To uninstall Falcon Operator run
   ```
   operator-sdk cleanup falcon-operator --namespace falcon-operator
   ```
