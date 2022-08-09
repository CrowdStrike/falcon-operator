# Deployment Guide for GKE
This document will guide you through the installation of falcon-operator and deployment of either the:
- [FalconContainer](../../cluster_resources/container/README.md) custom resource to the cluster with Falcon Container image being mirrored from CrowdStrike container registry to GCR (Google Container Registry). A new GCP service account for pushing to GCR registry will be created.
- [FalconNodeSensor](../../cluster_resources/node/README.md) custom resource to the cluster.

## Prerequisites

- CrowdStrike CWP subscription
- If your are installing the CrowdStrike Sensor via the Crowdstrike API, you need to create a new CrowdStrike API key pair with the following permissions:
  - Falcon Images Download: Read
  - Sensor Download: Read

## Installing the operator

- Either spin up an GKE Kubernetes cluster or use one that already exists.
- Install the operator
  ```
  kubectl apply -f https://raw.githubusercontent.com/CrowdStrike/falcon-operator/main/deploy/falcon-operator.yaml
  ```

### Deploy the Node Sensor

Once the operator has deployed, you can now deploy the FalconNodeSensor.

- Deploy FalconNodeSensor through the cli using the `kubectl` command:
  ```
  kubectl create -n falcon-operator -f https://raw.githubusercontent.com/CrowdStrike/falcon-operator/main/config/samples/falcon_v1alpha1_falconnodesensor.yaml --edit=true
  ```

### Deploying the Node Sensor to a custom Namespace

If desired, the FalconNodeSensor can be deployed to a namespace of your choosing instead of deploying to the operator namespace.
To deploy to a custom namespace (replacing `falcon-system` as desired):

- Create a new project
  ```
  kubectl create namespace falcon-system
  ```

- Create the service account in the new namespace
  ```
  kubectl create sa falcon-operator-node-sensor -n falcon-system
  ```

- Deploy FalconNodeSensor to the custom namespace:
  ```
  kubectl create -n falcon-system -f https://raw.githubusercontent.com/CrowdStrike/falcon-operator/main/docs/config/samples/falcon_v1alpha1_falconnodesensor.yaml --edit=true
  ```

### Deploy the sidecar sensor

#### Create GCR push secret

An image push secret is used by the operator to mirror Falcon Container image from CrowdStrike registry to your GCR.

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

- Store the newly created private key for image push in kubernetes
  ```
  kubectl create secret docker-registry -n falcon-system-configure builder --from-file .dockerconfigjson
  ```

#### Create the FalconContainer resource

- Create new FalconContainer resource
  ```
  kubectl create -f https://raw.githubusercontent.com/CrowdStrike/falcon-operator/main/docs/deployment/gke/falconcontainer.yaml --edit=true
  ```

#### Complete install using GCP Cloud Shell

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

## Uninstalling

When uninstalling the operator, it is important to make sure to uninstall the deployed custom resources first *before* you uninstall the operator.
This will insure proper cleanup of the resources.

### Uninstall the Node Sensor

- To uninstall the node sensor, simply remove the FalconNodeSensor resource.
  ```
  kubectl delete falconnodesensor -A --all
  ```

### Uninstall the Sidecar Sensor

- To uninstall Falcon Container, simply remove the FalconContainer resource. The operator will then uninstall the Falcon Container product from the cluster.
  ```
  kubectl delete falconcontainers --all
  ```

### Uninstall the Operator

- To uninstall Falcon Operator, delete the deployment:
  ```
  kubectl delete -f https://raw.githubusercontent.com/CrowdStrike/falcon-operator/main/deploy/falcon-operator.yaml
  ```

## GKE Node Upgrades

If the sidecar sensor has been deployed to your GKE cluster, you will want to explicitly disable CrowdStrike Falcon from monitoring using labels for the kube-public, kube-system, falcon-operator, and falcon-system namespaces.
For example:
```
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

```
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

- Delete the previously deployed FalconContainer resource:
  ```
  kubectl delete falconcontainers --all
  ```

- Add the newly created Service Account to your FalconContainer yaml file:
  ```
  spec:
    injector:
      sa_annotations:
        iam.gke.io/gcp-service-account: $GCP_SERVICE_ACCOUNT@$GCP_PROJECT_ID.iam.gserviceaccount.com
  ```

  Do not forget to replace the service account name template with actual name
  ```
  echo "$GCP_SERVICE_ACCOUNT@$GCP_PROJECT_ID.iam.gserviceaccount.com"
  ```

- Deploy the FalconContainer resource with the IAM role changes:
  ```
  kubectl create -f ./my-falcon-container.yaml
  ```