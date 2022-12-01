[#](#) Deployment Guide for OpenShift
This document will guide you through the installation of falcon-operator and deployment of [FalconContainer](../../container) custom resource to the cluster with Falcon Container image being mirrored from CrowdStrike container registry to OpenShift ImageStreams (on cluster registry).

## Pre-requisites

 - Have CrowdStrike CWP subscription with Falcon Container enabled (have your CrowdStrike CID ready)
 - Create new CrowdStrike API key pair with permission to download the sensor (no other permission shall be required)

## Installation Steps

 - Authenticate to your OpenShift cluster
   ```
   oc login --token=sha256~abcde-ABCDE-1 --server=https://openshift.example.com
   ```

 - Create namespaces used by the operator
   ```
   oc create ns falcon-operator --dry-run=client -o yaml | oc apply -f -
   oc create ns falcon-system --dry-run=client -o yaml | oc apply -f -
   ```
 
 - Add Falcon Operator subscription to the operator hub on the cluster (This is needed until falcon-operator is available through operatorhub.io)
   ```
   operator-sdk run bundle quay.io/crowdstrike/falcon-operator-bundle:latest --namespace falcon-operator
   ```

 - Deploy FalconContainer either through OpenShift web console or through `oc`
   ```
   oc create -f https://raw.githubusercontent.com/CrowdStrike/falcon-operator/main/docs/deployment/openshift/falconcontainer-openshift.yaml --edit=true
   ```
   
## Uninstall Steps

 - To uninstall Falcon Container simply remove FalconContainer resource. The operator will uninstall Falcon Container product from the cluster.
   ```
   oc delete falconcontainers.falcon.crowdstrike.com default
   ```
 - To uninstall Falcon Operator run
   ```
   operator-sdk cleanup falcon-operator --namespace falcon-operator
   ```
