<!--- NOTE: DO NOT EDIT! This file is auto-generated. Please update the source *.tmpl file instead --->
# Deployment Guide for GKE and GCR
This document will guide you through the installation of the Falcon Operator and deployment of the following custom resources provided by the Falcon Operator:
- [FalconAdmission](../../resources/admission/README.md) with the Falcon Admission Controller image being mirrored from CrowdStrike container registry to GCR (Google Container Registry). A new GCP service account for pushing to GCR registry will be created.
- [FalconContainer](../../resources/container/README.md) with the Falcon Container image being mirrored from CrowdStrike container registry to GCR (Google Container Registry). A new GCP service account for pushing to GCR registry will be created.
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

</details>

### Deploying the Falcon Admission Controller

<details>
  <summary>Click to expand</summary>

- Create a new FalconAdmission resource
  ```sh
  kubectl create -f https://raw.githubusercontent.com/crowdstrike/falcon-operator/main/docs/deployment/gke/falconadmission.yaml --edit=true
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

## GKE Autopilot configuration

<details>
  <summary>Click to expand</summary>

### Configuring the AllowlistSynchronizer
Running Daemonset Pods with privileged access on GKE Autopilot requires special configurations due to default security restrictions. To enable these privileged Pods, you need to configure an AllowlistSynchronizer. This resource applies CrowdStrike specific WorkloadAllowlists to your cluster, which the GKE Autopilot validating webhook uses to approve Pod deployments based on their manifest spec and image digests. Follow these steps to properly configure the AllowlistSynchronizer:
Comment


1. Create a file named `allowlist-synchronizer.yaml` with the following contents:
```
apiVersion: auto.gke.io/v1
kind: AllowlistSynchronizer
metadata:
  name: crowdstrike-synchronizer
spec:
  allowlistPaths:
  - CrowdStrike/falcon-sensor/*
```
2. Apply the AllowlistSynchronizer to your cluster:
```
kubectl apply -f allowlist-synchronizer.yaml
```

3. Ensure the AllowlistSynchronizer is running:
```
kubectl get allowlistsynchronizers
```

4. Ensure the AllowlistSynchronizer has fetched the WorkloadAllowlist:
```
kubectl get workloadallowlists
```
An example output of the above command is:
```
NAME                                                  AGE
crowdstrike-falconsensor-cleanup-allowlist-v1.0.0     7d
crowdstrike-falconsensor-cleanup-allowlist-v1.0.1     7d
crowdstrike-falconsensor-cleanup-allowlist-v1.0.2     7d
crowdstrike-falconsensor-deploy-allowlist-v1.0.0      7d
crowdstrike-falconsensor-deploy-allowlist-v1.0.1      7d
crowdstrike-falconsensor-deploy-allowlist-v1.0.2      7d
crowdstrike-falconsensor-deploy-allowlist-v1.0.3      6h40m
crowdstrike-falconsensor-falconctl-allowlist-v1.0.0   7d
crowdstrike-falconsensor-falconctl-allowlist-v1.0.1   7d
```
#### WorkloadAllowlist Definitions
The WorkloadAllowlists serve the following purposes:
- crowdstrike-falconsensor-cleanup-allowlist-vX.X.X: Authorizes the Falcon Sensor Cleanup DaemonSet to operate within the cluster.
- crowdstrike-falconsensor-deploy-allowlist-vX.X.X: Permits the deployment and execution of the Falcon Sensor Deploy DaemonSet in the cluster environment.
- crowdstrike-falconsensor-falconctl-allowlist-vX.X.X: Enables the Falconctl job to run, facilitating sensor configuration and management tasks.

> [!NOTE]
> Additional information about AllowlistSynchronizer can be found here: [https://cloud.google.com/kubernetes-engine/docs/reference/crds/allowlistsynchronizer](https://cloud.google.com/kubernetes-engine/docs/reference/crds/allowlistsynchronizer)

#### Obtaining an Authorized Image
WorkloadAllowlists ensure that only authorized container images are deployed to pods by verifying their image digests. To view the list of approved image digests, execute the following command:
```
kubectl get workloadallowlists <crowdstrike-falconsensor-XXXXXXX-allowlist-vX.X.X>  -o=jsonpath='{range .containerImageDigests[*].imageDigests[*]}{@}{"\n"}{end}'
```
To obtain the Falcon Node sensor image, you have two options:

1. Pull directly from the CrowdStrike registry
2. Copy the image from the CrowdStrike registry to your private registry

For option 2, we provide an automation script to simplify the process:
[https://github.com/CrowdStrike/falcon-scripts/tree/main/bash/containers/falcon-container-sensor-pull](https://github.com/CrowdStrike/falcon-scripts/tree/main/bash/containers/falcon-container-sensor-pull)

When copying images to a private registry, it's crucial to preserve the image digest. We recommend using tools like Skopeo for this purpose, as they ensure the digest of the image remains the same after the transfer. Additionally, the usage of an image digest should be used in the deployment manifest when an private registry is used. Example:
```
apiVersion: falcon.crowdstrike.com/v1alpha1
kind: FalconNodeSensor
metadata:
  name: falcon-node-sensor
spec:
  falcon_api:
    client_id: <client_id>
    client_secret: <client_secret>
    cloud_region: autodiscover
node:
  image: private.registry/falcon-sensor:sha256:ef5b80182894bba37c23aeea2748683bde186914b28e193708e6919c2549d396
  imagePullSecrets:
    - name: internal-registry-secret
```

### Setting the PriorityClass

When you enable GKE Autopilot deployment in the Falcon Operator, this creates a new PriorityClass to ensure that the sensor DaemonSet has priority over other application pods. This means that it’s possible that some application pods are evicted or pushed back in the scheduling queue depending on cluster resources availability to accommodate sensor Pods. You can either allow the operator to deploy its own PriorityClass or specify an existing PriorityClass.

### Configuring the resource usage

GKE Autopilot enforces supported minimum and maximum values for the total resources requested by your deployment configuration and will adjust the limits and requests to be within the min/max range. GKE Autopilot lets you set requests but not limits, and will mutate the limits to match the request values.



```yaml
resources:
  requests:
    cpu: "250m"
  limits:
    cpu: "<mutates to match requests>"
```

To understand how GKE Autopilot adjusts limits, and the minimum and maximum resource requests, see [Google Cloud documentation: Minimum and maximum resource requests](https://cloud.google.com/kubernetes-engine/docs/concepts/autopilot-resource-requests#min-max-requests).

The Falcon sensor resource usage depends on application workloads and therefore requires more resources if the sensor observes more events. The sensor defaults defined for memory and CPU are only for a successful sensor deployment. Consider adjusting the sensor memory and CPU within the allowed range enforced by GKE Autopilot to ensure the sensor deploys correctly.

> [!WARNING]
> Incorrect resource settings can lead to sensor deployment failure or a loss of clouded events. Avoid the following:
> * Setting requests or limits that are too low
> * Setting requests and limits that do not match ([more info](https://cloud.google.com/kubernetes-engine/docs/concepts/autopilot-resource-requests#resource-limits))

If the sensor fails to start, it’s likely because the application workload requires more resources. If this is the case, set the resource requests to a value higher within the acceptable GKE Autopilot min/max range.
If you notice that the resource allocation is too high for the application workloads, lower the resource requests as needed.

You can retrieve a snapshot of your resource usage with `kubectl top`, or other resource monitoring like Datadog or Prometheus. For example, the following command will show your CPU and Memory resource usage in the `falcon-system` namespace.

```shell
kubectl top pod -n falcon-system
NAME                                   CPU(cores)   MEMORY(bytes)
falcon-helm-falcon-node-sensor-slsmg   498m         223Mi
```

The sensor resource limits are only enabled when `backend: bpf`, which is a requirement for GKE Autopilot.

### Enabling GKE Autopilot

<details>
  <summary>Click to expand</summary>

To enable GKE Autopilot and deploy the sensor running in user mode, configure the following settings:

1. Set the backend to run in user mode.
   ```yaml
   node:
     backend: bpf
   ```

2. Enable GKE Autopilot.
   ```yaml
   node:
     gke:
       autopilot: true
   ```

3. Optionally, provide a name for an existing priority class, or the operator will create one for you.
   ```yaml
   node:
     priorityClass:
       Name: my_custom_priorityclass
   ```

4. Based on your workload requirements, set the requests and limits. The default values for GKE Autopilot are `750m` CPU and `1.5Gi` memory. The minimum allowed values are `250m` CPU and `500Mi` memory:
   ```yaml
   node:
     resources:
       cpu: 750m
       memory: 1.5Gi
   ```
   > [!WARNING]
   > If you set the requests or limits too low, you can potentially cause the sensor deployment to fail or cause loss of clouded events.

Add the following toleration to deploy correctly on autopilot:

```yaml
    - effect: NoSchedule
      key: kubernetes.io/arch
      operator: Equal
      value: amd64
```

Putting it altogether, an example completed node sensor configuration for GKE Autopilot could look like the following:

```yaml
node:
  backend: bpf
  gke:
    autopilot: true
  resources:
    requests:
      cpu: 750m
      memory: 1.5Gi
  tolerations:
    - effect: NoSchedule
      operator: Equal
      key: kubernetes.io/arch
      value: amd64
```

</details>

</details>

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

## Enabling GCP Workload Identity

### Enabling GCP Workload Identity for the Falcon Sidecar Injector

<details>
  <summary>Click to expand</summary>

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

#### Assigning GCP Workload Identity to Falcon Container Injector

<details>
  <summary>Click to expand</summary>

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

- Allow Falcon Sidecar Injector to use the newly created GCP Service Account
  ```sh
  gcloud iam service-accounts add-iam-policy-binding \
      $GCP_SERVICE_ACCOUNT@$GCP_PROJECT_ID.iam.gserviceaccount.com \
      --role roles/iam.workloadIdentityUser \
      --member "serviceAccount:$GCP_PROJECT_ID.svc.id.goog[falcon-system/falcon-operator-sidecar-sensor]"
  ```

- Delete the previously deployed FalconContainer resource:
  ```sh
  kubectl delete falconcontainers --all
  ```

- Add the newly created Service Account to your FalconContainer yaml file:
  ```yaml
  spec:
    injector:
      annotations:
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

</details>
</details>

### Enabling GCP Workload Identity for the Falcon Admission Controller

<details>
  <summary>Click to expand</summary>

The Falcon Admission Controller may need [GCP Workload Identity](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity)
to read GCR or Artifact Registry. In many cases, the GCP Workload Identity is assigned or inherited automatically.
Conceptually, the following tasks need to be done in order to enable GCR to pull from the injector:

- Create GCP Service Account
- Grant GCR permissions to the newly created Service Account
- Allow Falcon Admission Controller to use the newly created Service Account
- Put GCP Service Account handle into your Falcon Admission resource for re-deployments

#### Assigning GCP Workload Identity to Falcon Admission Controller

<details>
  <summary>Click to expand</summary>

Using both `gcloud` and `kubectl` command-line tools, perform the following steps:

- Set up your shell environment variables
  ```sh
  GCP_SERVICE_ACCOUNT=falcon-admission-controller

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

- Allow Falcon Admission Controller to use the newly created GCP Service Account
  ```sh
  gcloud iam service-accounts add-iam-policy-binding \
      $GCP_SERVICE_ACCOUNT@$GCP_PROJECT_ID.iam.gserviceaccount.com \
      --role roles/iam.workloadIdentityUser \
      --member "serviceAccount:$GCP_PROJECT_ID.svc.id.goog[falcon-kac/falcon-operator-admission-controller]"
  ```

- Delete the previously deployed FalconAdmission resource:
  ```sh
  kubectl delete falconadmission --all
  ```

- Add the newly created Service Account to your FalconAdmission yaml file:
  ```yaml
  spec:
    admissionConfig:
      annotations:
        iam.gke.io/gcp-service-account: $GCP_SERVICE_ACCOUNT@$GCP_PROJECT_ID.iam.gserviceaccount.com
  ```

  Do not forget to replace the service account name template with actual name
  ```sh
  echo "$GCP_SERVICE_ACCOUNT@$GCP_PROJECT_ID.iam.gserviceaccount.com"
  ```

- Deploy the FalconAdmission resource with the IAM role changes:
  ```sh
  kubectl create -f ./my-falcon-admission.yaml
  ```

</details>
</details>

## Troubleshooting
### GKE AutoPilot Installation Failures

<details>
  <summary>Click to expand</summary>

If your cluster utilizes AllowListSynchronizers and WorkloadAllowlists, you may run into issues with the GKE AutoPilot validating webhooks. An example of such an error in the controller-manager logs is:
```
2025-05-07T16:36:52Z    ERROR    Failed to create new DaemonSet    {"controller": "falconnodesensor", "controllerGroup": "falcon.crowdstrike.com", "controllerKind": "FalconNodeSensor", "FalconNodeSensor": {"name":"falcon-node-sensor"}, "namespace": "", "name": "falcon-node-sensor", "reconcileID": "b9d23c0e-fff5-41ca-9457-a8692d59794e", "DaemonSet": {"name":"falcon-node-sensor"}, "error": "admission webhook \"warden-validating.common-webhooks.networking.gke.io\" denied the request: \n\n=======================================================================\nWorkload Mismatches Found for Allowlist (crowdstrike-falconsensor-deploy-allowlist-v1.0.2):\n=======================================================================\nContainer[0] (falcon-node-sensor):\n- EnvFrom Mismatch: Verify that both configMapRef & secretRef match allowlist value."}
```
WorkloadAllowlists can be retrieved with the following:
```
kubectl get workloadallowlists
```

Specific WorkloadAllowlists can be used for troubleshooting. For example, if you are having issues with the `falcon-node-sensor` DaemonSet, you can run the FalconNodeSensor deployment against the `crowdstrike-falconsensor-deploy-allowlist-v1.0.2` WorkloadAllowlist.
```
spec:
  node:
    gke:
      autopilot: true
      deployAllowListVersion: v1.0.2
```

Failures encountered with WorkloadAllowlist for the cleanup daemonset may require manual intervention to fully uninstall. If the daemonset gets stuck in a continuous loop, complete the following:
```
# Get the cleanup daemonset name
kubectl get ds -n <namespace>

# Remove the Finalizer
kubectl patch daemonset <daemonset-name> -n <namespace> -p '{"metadata":{"finalizers":[]}}' --type=merge

# Delete the FalconNodeSensor
kubectl delete falconnodesensors --all
```
</details>
