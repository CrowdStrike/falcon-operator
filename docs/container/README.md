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
  installer_args:
    - '-falconctl-opts'
    - '--tags=test-cluster'
```

### Installation Phases

Once the FalconContainer resource is pushed to the cluster the operator will start an installation process. The installation process consists of the following 5 phases

| Phase         | Description                                                                                                                                  |
| :-------------| :--------------------------------------------------------------------------------------------------------------------------------------------|
| *Pending*     | Namespace `falcon-system-configure` is created. Optionally registry may be initialised (OpenShift ImageStream or new ECR repository created) |
| *Building*    | Falcon Container is pushed to custom registry (not applicable if `registry.type=crowdstrike` that skips the image push)                      |
| *Configuring* | Falcon Container Installer is run in `falcon-system-configure` namespace as Kubernetes Job. Operator waits for the Job completion            |
| *Deploying*   | Using the Installer output, Falcon Container is installed to the cluster                                                                     |
| *Validating*  | Operator asserts whether Falcon Container is deployed successfully                                                                           |
| *Done*        | Falcon Container Injector is up and running in `falcon-system` namespace.                                                                    |

### FalconContainer Reference Manual

| Spec                                | Description                                                                                                                              |
| :---------------------------------- | :----------------------------------------------------------------------------------------------------------------------------------------|
| falcon_api.client_id                | CrowdStrike API Client ID                                                                                                                |
| falcon_api.client_secret            | CrowdStrike API Client Secret                                                                                                            |
| falcon_api.client_region            | CrowdStrike cloud region (allowed values: autodiscover, us-1, us-2, eu-1, us-gov-1)                                                      |
| registry.type                       | Registry to mirror Falcon Container (allowed values: acr, ecr, crowdstrike, gcr, openshift))                                             |
| registry.tls.insecure_skip_verify   | (optional) Skip TLS check when pushing Falcon Container to target registry (only for demoing purposes on self-signed openshift clusters) |
| registry.acr_name                   | (optional) Name of ACR for the Falcon Container push. Only applicable to Azure cloud. (`registry.type="acr"`)                            |
| installer_args                      | (optional) Additional arguments to Falcon Container Installer (see [Product Documentation](https://falcon.crowdstrike.com/documentation/146/falcon-container-sensor-for-linux)) |
| version                             | (optional) Enforce particular Falcon Container version to be installed (example: "6.31", "6.31.0", "6.31.0-1409")                        | 

| Status                              | Description                                                                                                                               |
| :---------------------------------- | :---------------------------------------------------------------------------------------------------------------------------------------- |
| phase                               | Current phase of the deployment (see Installation Phases above) Must be `DONE` on successful deployment)                                  |
| errormsg                            | Displays the last notable error. Must be empty on successful deployment.                                                                  |
| version                             | Version of Falcon Container that is currently deployed                                                                                    |
| retry_attempt                       | Number of previous failed attempts (valid values: 0-5)                                                                                    |
| conditions.["ImageReady"]           | Informs about readiness of Falcon Container image. Custom message refers to image URI that will be used during the deployment             |
| conditions.["InstallerComplete"]    | Informs about completion of Falcon Container Installer. Users can review the installer Job/Pod in `falcon-system-configure` namespace     |
| conditions.["Complete"]             | Informs about the completion of the deployment of Falcon Container                                                                        |

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
| falcon-system-configure | Used by operator, contains objects created by operator           |

### Compatibility Guide

Falcon Operator has been explicitly tested on AKS (with ECR), EKS (with ECR), GKE (with GCR), and OpenShift (with ImageStreams).

| Platform                      | Supported versions                                     |
|:------------------------------|:-------------------------------------------------------|
| AKS (with ACR)                | 1.18 or greater                                        |
| EKS (with ECR)                | 1.17 or greater                                        |
| GKE (with GCR)                | 1.18 or greater                                        |
| OpenShift (with ImageStreams) | 4.7 or greater                                         |

### Troubleshooting

Falcon Operator modifies the FalconContainer CRD based on what is happening in the cluster. Should an error occur during Falcon Container deployment that error will appear in kubectl output as shown below.

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

To review the logs of Falcon Container Installer:
```
kubectl logs -n falcon-system-configure job/falcon-configure
```

To review the logs of Falcon Container Injector:
```
kubectl logs -n falcon-system deploy/injector -f
```

### Additional Documentation
End-to-end guides to install Falcon-operator together with FalconContainer resource.

 - [Deployment Guide for EKS/ECR](../../docs/deployment/eks/README.md)
 - [Deployment Guide for GKE/GCR](../../docs/deployment/gke/README.md)
 - [Deployment Guide for OpenShift](../../docs/deployment/openshift/README.md)
