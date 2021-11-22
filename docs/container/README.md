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


## Installation Steps
Falcon Operator provides automated install & uninstall of a Falcon Container Sensor. To start a new installation please push the FalconContainer resource to your cluster. A sample FalconContainer resource follows:

```
apiVersion: falcon.crowdstrike.com/v1alpha1
kind: FalconContainer
metadata:
  name: default
spec:
  falcon_api:
    client_id: PLEASE_FILL_IN
    client_secret: PLEASE_FILL_IN
    cloud_region: us-1
  registry:
    type: crowdstrike
  installer_args:
    - '-falconctl-opts'
    - '--tags=test-cluster'
```

The `cid` parameter refers to CrowdStrike Customer ID. This CID will be used to start Falcon Container sensors and all the data will be reported to that CID. The `client_id` and `client_secret` parameters refer to API key pairs used to download the CrowdStrike Falcon Container sensor (no other permission except the sensor download shall be granted to this API key pair).

When FalconContainer resources are pushed to the cluster, falcon-operator will automatically install the Falcon Container product to the cluster.

### Uninstall Steps
 - To uninstall Falcon Container simply remove the FalconContainer resource. The operator will uninstall the Falcon Container product from the cluster.

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

Falcon Operator supports EKS (with ECR), GKE (with GCR), and OpenShift (with ImageStreams).

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
 - [Deployment Guide for EKS/ECR](../../docs/deployment/eks/README.md)
 - [Deployment Guide for GKE/GCR](../../docs/deployment/gke/README.md)
 - [Deployment Guide for OpenShift](../../docs/deployment/openshift/README.md)
