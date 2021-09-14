# Deployment Guide for OpenShift

## Pre-requisites

 - Have CrowdStrike CWP subscription with Falcon Container enabled (have your CrowdStrike CID ready)
 - Have Container Administrator access to GCP and at least one GKE cluster deployed
 - Create new CrowdStrike API key pair with permission to download the sensor (no other permission shall be required)

## Installation Steps

 - Authenticate to your OpenShift cluster
   ```
   oc login --token=sha256~abcde-ABCDE-1 --server=https://openshift.example.com
   ```

 - Create namespaces used by the operator
   ```
   kubectl create ns falcon-operator --dry-run=client -o yaml | kubectl apply -f
   kubectl create ns falcon-system-configure --dry-run=client -o yaml | kubectl apply -f -
   ```
 
 - Add Falcon Operator subscription to the operator hub on the cluster (This is needed until falcon-operator is available through operatorhub.io)
   ```
   operator-sdk run bundle quay.io/crowdstrike/falcon-operator-bundle --namespace falcon-operator
   ```

 - Deploy FalconContainer either through OpenShift web console or through `kubectl`
   ```
   kubectl create -f https://raw.githubusercontent.com/CrowdStrike/falcon-operator/main/docs/deployment/openshift/falconcontainer-openshift.yaml --edit=true
   ```
   
## Uninstall Steps

 - To uninstall Falcon Container simply remove FalconContainer resource. The operator will uninstall Falcon Container product from the cluster.
   ```
   kubectl delete falconcontainers.falcon.crowdstrike.com default
   ```
 - To uninstall Falcon Operator run
   ```
   operator-sdk cleanup falcon-operator --namespace falcon-operator
   ```
