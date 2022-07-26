# Falcon Container Sensor


## About Falcon Container Sensor
The Falcon Container sensor for Linux extends runtime security to container workloads in Kubernetes clusters that don’t allow you to deploy the kernel-based Falcon sensor for Linux. The Falcon Container sensor runs as an unprivileged container in user space with no code running in the kernel of the worker node OS. This allows it to secure Kubernetes pods in clusters where it isn’t possible to deploy the kernel-based Falcon sensor for Linux on the worker node, as with AWS Fargate where organizations don’t have access to the kernel and where privileged containers are disallowed. The Falcon Container sensor can also secure container workloads on clusters where worker node security is managed separately.

### Core Features
 - **Leverage market-leading protection technologies:** Machine learning (ML), artificial intelligence (AI), indicators of attack (IOAs) and custom hash blocking automatically defend against malware and sophisticated threats targeting containers.
 - **Stop malicious behavior:** Behavioral profiling enables you to block activities that violate policy with zero impact to legitimate container operation.
 - **Investigate container incidents faster:** Easily investigate incidents when detections are associated with the specific container and not bundled with host events.
 - **See everything:** Capture container start, stop, image, runtime information and all events generated inside each and every container.
 - **Deploy seamlessly with Kubernetes:** Deploy easily at scale by including it as part of a Kubernetes cluster.
 - **Improve container orchestration:** Capture Kubernetes namespace, pod metadata, process, file and network events.

Learn more at [product pages](https://www.crowdstrike.com/products/cloud-security/falcon-cloud-workload-protection/container-security/).


## About FalconContainer Custom Resource
Falcon Operator introduces FalconContainer Custom Resource to the cluster. The resource is meant to be singleton and it will install, configure and uninstall Falcon Container Sensor on the cluster.

To start the Falcon Container installation please push the following FalconContainer resource to your cluster. You will need to provide CrowdStrike API Keys and CrowdStrike cloud region for the installation. It is recommended to establish new API credentials for the installation at https://falcon.crowdstrike.com/support/api-clients-and-keys, minimal required permissions are:
 * Falcon Images Download: Read
 * Sensor Download: Read

No other permissions shall be granted to the new API key pair.

```
apiVersion: falcon.crowdstrike.com/v1alpha1
kind: FalconContainer
metadata:
  name: default
spec:
  falcon:
    Tags: 'test-cluster,dev'
  falcon_api:
    client_id: PLEASE_FILL_IN
    client_secret: PLEASE_FILL_IN
    cloud_region: autodiscover
  registry:
    type: crowdstrike
```

### FalconContainer Reference Manual

| Spec                                      | Description                                                                                                                                                                                                             |
| :----------------------------------       | :----------------------------------------------------------------------------------------------------------------------------------------                                                                               |
| falcon_api.client_id                      | CrowdStrike API Client ID                                                                                                                                                                                               |
| falcon_api.client_secret                  | CrowdStrike API Client Secret                                                                                                                                                                                           |
| falcon_api.client_region                  | CrowdStrike cloud region (allowed values: autodiscover, us-1, us-2, eu-1, us-gov-1)                                                                                                                                     |
| falcon.apd                                | (optional) Configure Falcon Sensor to leverage a proxy host                                                                                                                                                             |
| falcon.aph                                | (optional) Configure the host Falcon Sensor should leverage for proxying                                                                                                                                                |
| falcon.app                                | (optional) Configure the port Falcon Sensor should leverage for proxying                                                                                                                                                |
| falcon.billing                            | (optional) Configure Pay-as-You-Go (metered) billing rather than default billing                                                                                                                                        |
| falcon.provisioning_token                 | (optional) Configure a Provisioning Token for CIDs with restricted AID provisioning enabled                                                                                                                             |
| falcon.tags                               | (optional) Configure Falcon Sensor Grouping Tags; comma-delimited                                                                                                                                                       |
| falcon.trace                              | (optional) Configure Falcon Sensor Trace Logging Level (none, err, warn, info, debug)                                                                                                                                   |
| image                                     | (optional) Leverage a Falcon Container Sensor image that is not managed by the operator; typically used with custom repositories; overrides all registry settings; might require injector.imagePullSecretName to be set |
| version                                   | (optional) Enforce particular Falcon Container version to be installed (example: "6.31", "6.31.0", "6.31.0-1409")                                                                                                       |
| versionLocking                            | (optional) Enable/Disable version locking; if disabled, new image versions (matching any specified version string) will be leveraged for future pod injections upon controller reconciliation (default: true)
| registry.type                     | Registry to mirror Falcon Container (allowed values: acr, ecr, crowdstrike, gcr, openshift)                                              |
| registry.tls.insecure_skip_verify | (optional) Skip TLS check when pushing Falcon Container to target registry (only for demoing purposes on self-signed openshift clusters) |
| registry.tls.caCertificate | (optional) A string containing an optionally base64-encoded Certificate Authority Chain for self-signed TLS Registry Certificates
| registry.tls.caCertificateConfigMap | (optional) The name of a ConfigMap containing CA Certificate Authority Chains under keys ending in ".tls"  for self-signed TLS Registry Certificates (ignored when registry.tls.caCertificate is set)
| registry.acr_name                         | (optional) Name of ACR for the Falcon Container push. Only applicable to Azure cloud. (`registry.type="acr"`)                                                                                                           |
| registry.ecr_iam_role_arn                 | (optional) ARN of AWS IAM Role to be assigned to the Injector (only needed when injector runs on EKS Fargate)                                                                                                           |
| injector.serviceAccount.name              | (optional) Name of Service Account to create in falcon-system namespace                                                                                                                                                 |
| injector.serviceAccount.annotations       | (optional) Annotations that should be added to the Service Account (e.g. for IAM role association)                                                                                                                      |
| injector.listenPort                       | (optional) Override the default Injector Listen Port of 4433                                                                                                                                                            |
| injector.replicas                       | (optional) Override the default Injector Replica count of 2                                                                                                                                                            |
| injector.tls.validity                     | (optional) Override the default Injector CA validity of 3650 days                                                                                                                                                       |
| injector.imagePullPolicy                  | (optional) Override the default Falcon Container image pull policy of Always                                                                                                                                            |
| injector.imagePullSecretName              | (optional) Provide a secret containing an alternative pull token for the Falcon Container image                                                                                                                         |
| injector.logVolume                        | (optional) Provide a volume for Falcon Container logs                                                                                                                                                                   |
| injector.resources                        | (optional) Provide a set of kubernetes resource requirements for the Falcon Injector                                                                                                                                    |
| injector.sensorResources                  | (optional) Provide a set of kubernetes resource requirements for the Falcon Container Sensor container                                                                                                                  |
| injector.additionalEnvironmentVariables   | (optional) Provide additional environment variables for Falcon Container                                                                                                                                                |
| injector.disableDefaultNamespaceInjection | (optional) If set to true, disables default Falcon Container injection at the namespace scope; namespaces requiring injection will need to be labeled as specified below                                                |
| injector.disableDefaultPodInjection       | (optional) If set to true, disables default Falcon Container injection at the pod scope; pods requiring injection will need to be annotated as specified below                                                          |

| Status                              | Description                                                                                                                               |
| :---------------------------------- | :---------------------------------------------------------------------------------------------------------------------------------------- |
| phase                               | Current phase of the deployment; either RECONCILING, ERROR, or DONE
| errormsg                                         | Displays the last notable error. Must be empty on successful deployment.                                                                                             |
| version                                          | Version of Falcon Container that is currently deployed                                                                                                               |
| conditions.["NamespaceReady"]                    | Displays the most recent reconciliation operation for the Namespace used by the Falcon Container Sensor (Created, Updated, Deleted)                                  |
| conditions.["ImageReady"]                        | Informs about readiness of Falcon Container image. Custom message refers to image URI that will be used during the deployment (Pushed, Discovered)                   |
| conditions.["ImageStreamReady"]                  | Displays the most recent successful reconciliation operation for the image stream used by the falcon container in openshift environments (created, updated, deleted) |
| conditions.["ServiceAccountReady"]               | Displays the most recent sucreconciliation operation for the service account used by the falcon container (created, updated, deleted)                                   |
| conditions.["ClusterRoleReady"]                  | Displays the most recent sucreconciliation operation for the cluster role used by the falcon container sensor (created, updated, deleted)                               |
| conditions.["ClusterRoleBindingReady"]           | Displays the most recent sucreconciliation operation for the cluster role binding used by the falcon container sensor (created, updated, deleted)                       |
| conditions.["SecretReady"]                       | Displays the most recent sucreconciliation operation for the secrets used by the falcon container sensor (created, updated, deleted)                                    |
| conditions.["ConfigMapReady"]                    | Displays the most recent sucreconciliation operation for the config map used by the falcon container sensor (created, updated, deleted)                                 |
| conditions.["DeploymentReady"]                   | Displays the most recent sucreconciliation operation for the deployment used by the falcon container sensor injector (created, updated, deleted)                        |
| conditions.["ServiceReady"]                      | Displays the most recent sucreconciliation operation for the service used by the falcon container sensor injector (created, updated, deleted)                           |
| conditions.["MutatingWebhookConfigurationReady"] | Displays the most recent sucreconciliation operation for the mutating webhook configuration used by the falcon container sensor injector (created, updated, deleted)    |

### Enabling and Disabling Falcon Container injection

By default, all pods in all namespaces outside of kube-system and kube-public will be subject to Falcon Container injection.

To disable sensor injection for all pods in one namespace, add a label to the namespace:
sensor.falcon-system.crowdstrike.com/injection=disabled

If injector.disableDefaultNamespaceInjection is set to true, then sensor injection will be disabled in all namespaces by default; to enable injection for all pods in one namespace with default namespace injection set to true, add a label to the namespace:
sensor.falcon-system.crowdstrike.com/injection=enabled


To disable sensor injection for one pod, add an annotation to the pod spec:
sensor.falcon-system.crowdstrike.com/injection=disabled

If injector.disableDefaultPodInjection is set to true, then sensor injection will be disabled for all pods by default; to enable injection for one pod in a namespace subject to injection, add an annotation to the pod spec:
sensor.falcon-system.crowdstrike.com/injection=enabled
 

### Image Registry considerations

Falcon Container Image is distributed by CrowdStrike through CrowdStrike Falcon registry. Operator supports two modes of deployment:

#### (Option 1) Use CrowdStrike registry directly

Does not require any advanced set-ups. Users are advised to use the following excerpt in theirs FalconContainer custom resource definition.

```
registry:
  type: crowdstrike
```

Falcon Container product will then be installed directly from CrowdStrike registry. Any new deployment to the cluster may contact CrowdStrike registry for the image download. The `falcon-crowdstrike-pull-secret imagePullSecret` is created in all the namespaces targeted for injection.

#### (Option 2) Let operator mirror Falcon Container image to your local registry

Requires advanced set-up to grant the operator push access to your local registry. The operator will then mirror Falcon Container image from CrowdStrike registry to your local registry of choice.

Supported registries are: acr, ecr, gcr, and openshift. Each registry type requires advanced set-up enable image push.

Consult specific deployment guides to learn about the steps needed for image mirroring.

 - [Deployment Guide for AKS/ACR](../../docs/deployment/azure/README.md)
 - [Deployment Guide for EKS/ECR](../../docs/deployment/eks/README.md)
 - [Deployment Guide for EKS Fargate](../../deployment/eks-fargate/README.md)
 - [Deployment Guide for GKE/GCR](../../docs/deployment/gke/README.md)
 - [Deployment Guide for OpenShift](../../docs/deployment/openshift/README.md)

#### (Option 3) Use a custom Image URI

Image must be available at the specified URI; setting the image attribute will cause registry settings to be ignored.  No image mirroring will be leveraged, and version locking is implied.

Example:
```
image: myprivateregistry.internal.lan/falcon-container/falcon-sensor:6.47.0-3003.container.x86_64.Release.US-1
```

### Install Steps
To install Falcon Container (assuming Falcon Operator is installed):
```
kubectl create -f https://raw.githubusercontent.com/CrowdStrike/falcon-operator/main/config/samples/falcon_v1alpha1_falconcontainer.yaml --edit=true
```

### Uninstall Steps
To uninstall Falcon Container simply remove the FalconContainer resource. The operator will uninstall the Falcon Container product from the cluster.

```
kubectl delete falconcontainers.falcon.crowdstrike.com --all
```

### Sensor Upgrades

The current version of the operator will update the Falcon Container Sensor version upon Operator Reconciliation, if Version Locking is disabled.  Note that this will only impact future Sensor injections, and will not cause any changes to running pods.  Version Locking is enabled by default. 

### Operator Upgrades

The current version of the operator does not support in place upgrades.  Users are advised to remove any FalconContainer or FalconNodeSensor Custom Resources, allow the operator to clean up previously deployed resources, uninstall the operator and install the target version, before updating their Custom Resource(s) to reflect any changes introduced ([see Release Notes](../../RELEASE.md)).

### Namespace Reference

The following namespaces will be used by Falcon Operator.

| Namespace               | Description                                                      |
|:------------------------|:-----------------------------------------------------------------|
| falcon-system           | Used by Falcon Container product, runs the injector and webhoook |
| falcon-operator         | Runs falcon-operator manager                                     |

### Compatibility Guide

Falcon Operator has been explicitly tested on AKS (with ECR), EKS (with ECR), GKE (with GCR), and OpenShift (with ImageStreams).

| Platform                      | Supported versions                                     |
|:------------------------------|:-------------------------------------------------------|
| AKS (with ACR)                | 1.18 or greater                                        |
| EKS (with ECR)                | 1.17 or greater                                        |
| GKE (with GCR)                | 1.18 or greater                                        |
| OpenShift (with ImageStreams) | 4.7 or greater                                         |

### Troubleshooting

Falcon Operator modifies the FalconContainer CR based on what is happening in the cluster. Should an error occur during Falcon Container deployment that error will appear in kubectl output as shown below.

```
$ kubectl get falconcontainers.falcon.crowdstrike.com
NAME      STATUS   VERSION                                     ERROR
default   DONE     6.31.0-1409.container.x86_64.Release.US-1
```

The empty ERROR column together with `status=DONE` indicates that Falcon Container deployment did not yield any errors. Should more insight be needed, users are advised to view FalconContainer CRD in full detail.

```
kubectl get falconcontainers.falcon.crowdstrike.com -o yaml
```

To review the logs of Falcon Operator:
```
kubectl -n falcon-operator logs -f deploy/falcon-operator-controller-manager -c manager
```

To review the logs of Falcon Container Injector:
```
kubectl logs -n falcon-system deploy/injector -f
```

To review the currently deployed version of the operator
```
kubectl get deployments -n falcon-operator falcon-operator-controller-manager -o=jsonpath='{.spec.template.spec.containers[].image}'
```

### Additional Documentation
End-to-end guides to install Falcon-operator together with FalconContainer resource.

 - [Deployment Guide for AKS/ACR](../../deployment/azure/README.md)
 - [Deployment Guide for EKS/ECR](../../deployment/eks/README.md)
 - [Deployment Guide for EKS Fargate](../../deployment/eks-fargate/README.md)
 - [Deployment Guide for GKE/GCR](../../deployment/gke/README.md)
 - [Deployment Guide for OpenShift](../../deployment/openshift/README.md)
