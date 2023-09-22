<!--- NOTE: DO NOT EDIT! This file is auto-generated. Please update the source *.tmpl file instead --->
# Deployment Guide for GKE and GCR
This document will guide you through the installation of the Falcon Operator and deployment of the following resources provdied by the Falcon Operator:
- [FalconContainer](../../resources/container/README.md) custom resource to the cluster with Falcon Container image being mirrored from CrowdStrike container registry to GCR (Google Container Registry). A new GCP service account for pushing to GCR registry will be created.
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
#### Create GCR push secret

An image push secret is used by the operator to mirror Falcon Container image from CrowdStrike registry to your GCR.

- Set environment variable to refer to your GCP project
  ```sh
  GCP_PROJECT_ID=$(gcloud config get-value core/project)
  ```

- Create new GCP service account
  ```sh
  gcloud iam service-accounts create falcon-operator
  ```

- Grant image push access to the newly created service account
  ```sh
  gcloud projects add-iam-policy-binding $GCP_PROJECT_ID \
      --member serviceAccount:falcon-operator@$GCP_PROJECT_ID.iam.gserviceaccount.com \
      --role roles/storage.admin
  ```

- Create new private key for the newly create service account
  ```sh
  gcloud iam service-accounts keys create \
      --iam-account "falcon-operator@$GCP_PROJECT_ID.iam.gserviceaccount.com" \
      .dockerconfigjson
  ```

- Store the newly created private key for image push in kubernetes
  ```
  kubectl create secret docker-registry -n falcon-system-configure builder --from-file .dockerconfigjson
  ```

#### Create the FalconContainer resource

- Create a new FalconContainer resource
  ```sh
  kubectl create -f https://raw.githubusercontent.com/crowdstrike/falcon-operator/main/docs/deployment/gke/falconcontainer.yaml --edit=true
  ```

#### Complete install using GCP Cloud Shell

- Open GCP Cloud Shell: https://shell.cloud.google.com/?hl=en_US&fromcloudshell=true&show=terminal
- Ensure the Cloud Shell is running in context of GCP project you want to use
  ```sh
  gcloud config get-value core/project
  ```
- In case you have multiple GKE clusters in the project, You need to select the desired one to install the operator in
  ```sh
  gcloud container clusters get-credentials DESIRED_CLUSTER --zone DESIRED_LOCATION
  ```
- Install the operator & operator-sdk & deploy Falcon Container Sensor
  ```sh
  bash -c 'source <(curl -s https://raw.githubusercontent.com/crowdstrike/falcon-operator/main/docs/deployment/gke/run)'
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



## GKE Node Upgrades

If the sidecar sensor has been deployed to your GKE cluster, you will want to explicitly disable CrowdStrike Falcon from monitoring using labels for the kube-public, kube-system, falcon-operator, and falcon-system namespaces.
For example:
```sh
kubectl label namespace falcon-operator sensor.falcon-system.crowdstrike.com/injection=disabled
kubectl label namespace falcon-system sensor.falcon-system.crowdstrike.com/injection=disabled
kubectl label namespace kube-system sensor.falcon-system.crowdstrike.com/injection=disabled
kubectl label namespace kube-public sensor.falcon-system.crowdstrike.com/injection=disabled
```

Because the Falcon Container sensor injector is configured to monitor all namespaces, setting the above labels will ensure that any pod related to k8 control plane and CrowdStrike Falcon are not forwarded to the injector.

## Granting GCP Workload Identity to Falcon Container Injector

The Falcon Container Injector may need [GCP Workload Identity](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity)
to read GCR or Artifact Registry. In many cases, the GCP Workload Identity is assigned or inherited automatically. However if you
are seeing errors similar to the following, you may need to follow this guide to assign the identity manually.

```log
time="2022-01-14T13:05:11Z" level=error msg="Failed to handle webhook request" error="Failed to retrieve image details for \"gcr.io/\" in container in pod: Failed to get the image config/digest for \"gcr.io/" on \"eu.gcr.io\": Error reading manifest latest in gcr.io/: unauthorized: You don't have the needed permissions to perform this operation, and you may have invalid credentials. To authenticate your request, follow the steps in: https://cloud.google.com/container-registry/docs/advanced-authentication"
```

Conceptually, the following tasks need to be done in order to enable GCR to pull from the injector:

- Create GCP Service Account
- Grant GCR permissions to the newly created Service Account
- Allow Falcon Container to use the newly created Service Account
- Put GCP Service Account handle into your Falcon Container resource for re-deployments

### Assigning GCP Workload Identity to Falcon Container Injector

Using both `gcloud` and `kubectl` command-line tools, perform the following steps:

- Set up your shell environment variables
  ```sh
  GCP_SERVICE_ACCOUNT=falcon-container-injector

  GCP_PROJECT_ID=$(gcloud config get-value core/project)
  ```

- Create new GCP Service Account
  ```sh
  gcloud iam service-accounts create $GCP_SERVICE_ACCOUNT
  ```

- Grant GCR permissions to the newly created Service Account
  ```sh
  gcloud projects add-iam-policy-binding $PROJECT_ID \
      --member "serviceAccount:$GCP_SERVICE_ACCOUNT@$GCP_PROJECT_ID.iam.gserviceaccount.com" \
      --role roles/containerregistry.ServiceAgent
  ```

- Allow Falcon Injector to use the newly created GCP Service Account
  ```sh
  gcloud iam service-accounts add-iam-policy-binding \
      $GCP_SERVICE_ACCOUNT@$GCP_PROJECT_ID.iam.gserviceaccount.com \
      --role roles/iam.workloadIdentityUser \
      --member "serviceAccount:$GCP_PROJECT_ID.svc.id.goog[falcon-system/default]"
  ```

- Delete the previously deployed FalconContainer resource:
  ```sh
  kubectl delete falconcontainers --all
  ```

- Add the newly created Service Account to your FalconContainer yaml file:
  ```yaml
  spec:
    injector:
      sa_annotations:
        iam.gke.io/gcp-service-account: $GCP_SERVICE_ACCOUNT@$GCP_PROJECT_ID.iam.gserviceaccount.com
  ```

  Do not forget to replace the service account name template with actual name
  ```sh
  echo "$GCP_SERVICE_ACCOUNT@$GCP_PROJECT_ID.iam.gserviceaccount.com"
  ```

- Deploy the FalconContainer resource with the IAM role changes:
  ```sh
  kubectl create -f ./my-falcon-container.yaml
  ```
