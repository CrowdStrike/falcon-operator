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
  falcon_api:
    client_id: PLEASE_FILL_IN
    client_secret: PLEASE_FILL_IN
    cloud_region: autodiscover
  registry:
    type: crowdstrike
  injector:
    falconctlOpts: '--tags=test-cluster'
```

Note: to provide multiple arguments to `falconctlOpts`, you need to provide them as a one line string:

```
  injector:
    falconctlOpts: `--tags=test-cluster,tags1,tags2 --apd=disabled`
```

### FalconContainer Reference Manual

| Spec                                | Description                                                                                                                              |
| :---------------------------------- | :----------------------------------------------------------------------------------------------------------------------------------------|
| falcon_api.client_id                | CrowdStrike API Client ID                                                                                                                |
| falcon_api.client_secret            | CrowdStrike API Client Secret                                                                                                            |
| falcon_api.client_region            | CrowdStrike cloud region (allowed values: autodiscover, us-1, us-2, eu-1, us-gov-1)                                                      |
| registry.type                       | Registry to mirror Falcon Container (allowed values: acr, ecr, crowdstrike, gcr, openshift)                                             |
| registry.tls.insecure_skip_verify   | (optional) Skip TLS check when pushing Falcon Container to target registry (only for demoing purposes on self-signed openshift clusters) |
| registry.acr_name                   | (optional) Name of ACR for the Falcon Container push. Only applicable to Azure cloud. (`registry.type="acr"`)                            |
| registry.ecr_iam_role_arn           | (optional) ARN of AWS IAM Role to be assigned to the Injector (only needed when injector runs on EKS Fargate)                            |
| injector.serviceAccount.name              | (optional) Name of Service Account to create in falcon-system namespace                                                                                                         |
| injector.serviceAccount.annotations       | (optional) Annotations that should be added to the Service Account (e.g. for IAM role association)                                                                              |
| injector.listenPort                       | (optional) Override the default Injector Listen Port of 4433                                                                                                                    |
| injector.tls.validity                     | (optional) Override the default Injector CA validity of 3650 days                                                                                                               |
| injector.imagePullPolicy                  | (optional) Override the default Falcon Container image pull policy of Always                                                                                                    |
| injector.imagePullSecretName              | (optional) Provide a secret containing an alternative pull token for the Falcon Container image                                                                                 |
| injector.logVolume                        | (optional) Provide a volume for Falcon Container logs                                                                                                                           |
| injector.resources                        | (optional) Provide a set of kubernetes resource requirements for the Falcon Injector                                                                                            |
| injector.sensorResources                  | (optional) Provide a set of kubernetes resource requirements for the Falcon Container Sensor container                                                                          |
| injector.falconctlOpts                    | (optional) Provide additional arguments to falconctl (e.g. '--tags myTestCluster')                                                                                              |
| injector.additionalEnvironmentVariables   | (optional) Provide additional environment variables for Falcon Container                                                                                                        |
| injector.disableDefaultNamespaceInjection | (optional) If set to true, disables default Falcon Container injection at the namespace scope; namespaces requiring injection will need to be labeled as specified below        |
| injector.disableDefaultPodInjection       | (optional) If set to true, disables default Falcon Container injection at the pod scope; pods requiring injection will need to be annotated as specified below                  |
| version                                   | (optional) Enforce particular Falcon Container version to be installed (example: "6.31", "6.31.0", "6.31.0-1409")                                                               |

| Status                              | Description                                                                                                                               |
| :---------------------------------- | :---------------------------------------------------------------------------------------------------------------------------------------- |
| phase                               | Current phase of the deployment; either RECONCILING, ERROR, or DONE
| errormsg                            | Displays the last notable error. Must be empty on successful deployment.                                                                  |
| version                             | Version of Falcon Container that is currently deployed                                                                                    |
| retry_attempt                       | Number of previous failed attempts (valid values: 0-5)                                                                                    |
| conditions.["ImageReady"]           | Informs about readiness of Falcon Container image. Custom message refers to image URI that will be used during the deployment             |
| conditions.["InstallerComplete"]    | Informs about completion of Falcon Container Installer. Users can review the installer Job/Pod in `falcon-system-configure` namespace     |
| conditions.["Complete"]             | Informs about the completion of the deployment of Falcon Container                                                                        |

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
 - [Deployment Guide for EKS/ECR](../../docs/deployment/eks/README.md) ([Fargate Considerations](../deployment/eks-fargate/README.md))
 - [Deployment Guide for GKE/GCR](../../docs/deployment/gke/README.md) ([GCP Workload Identity Considerations](../deployment/gke/gcp-workload-identity.md))
 - [Deployment Guide for OpenShift](../../docs/deployment/openshift/README.md)

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

### Upgrades

The current version of the operator does not automatically update Falcon Container sensor. Users are advised to remove & re-add FalconContainer resource to uninstall Falcon Container and to install the newest version.

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

 - [Deployment Guide for AKS/ACR](../../docs/deployment/azure/README.md)
 - [Deployment Guide for EKS/ECR](../../docs/deployment/eks/README.md) ([Fargate Considerations](../deployment/eks-fargate/README.md))
 - [Deployment Guide for GKE/GCR](../../docs/deployment/gke/README.md) ([GCP Workload Identity Considerations](../deployment/gke/gcp-workload-identity.md))
 - [Deployment Guide for OpenShift](../../docs/deployment/openshift/README.md)
