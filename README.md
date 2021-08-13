# Falcon Operator
[![Artifact Hub](https://img.shields.io/endpoint?url=https://artifacthub.io/badge/repository/falcon-operator)](https://artifacthub.io/packages/search?repo=falcon-operator)
[![CI Golang Build](https://github.com/CrowdStrike/falcon-operator/actions/workflows/go.yml/badge.svg)](https://github.com/CrowdStrike/falcon-operator/actions/workflows/go.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/crowdstrike/falcon-operator)](https://goreportcard.com/report/github.com/crowdstrike/falcon-operator)
[![gosec](https://github.com/CrowdStrike/falcon-operator/actions/workflows/gosec.yml/badge.svg)](https://github.com/CrowdStrike/falcon-operator/actions/workflows/gosec.yml)
[![CodeQL](https://github.com/CrowdStrike/falcon-operator/actions/workflows/codeql.yml/badge.svg)](https://github.com/CrowdStrike/falcon-operator/actions/workflows/codeql.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/crowdstrike/falcon-operator.svg)](https://pkg.go.dev/github.com/crowdstrike/falcon-operator)
[![CI Container Build](https://github.com/CrowdStrike/falcon-operator/actions/workflows/container_build.yml/badge.svg)](https://github.com/CrowdStrike/falcon-operator/actions/workflows/container_build.yml)
[![Docker Repository on Quay](https://quay.io/repository/crowdstrike/falcon-operator/status "Docker Repository on Quay")](https://quay.io/repository/crowdstrike/falcon-operator)
[![Docker Repository on Quay](https://quay.io/repository/crowdstrike/falcon-operator-bundle/status "Docker Repository on Quay")](https://quay.io/repository/crowdstrike/falcon-operator-bundle)

Falcon Operator installs CrowdStrike Falcon Container Sensor on the cluster.

Falcon Operator is an open source project, not CrowdStrike product. As such it carries no formal support, expressed or implied.

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

## About Falcon Operator
Falcon Operator deploys CrowdStrike Falcon Container Workload Protection the cluster. The operator introduces Custom Resource: FalconContainer that allows easy install & uninstall of the Falcon Container.

### Installation Steps
Falcon Operator provides automated install & uninstall of Falcon Container Sensor. To start new installation please push FalconContainer resource to your cluster. A sample FalconContainer resource follows:

```
apiVersion: falcon.crowdstrike.com/v1alpha1
kind: FalconContainer
metadata:
  name: default
spec:
  falcon_api:
    cid: PLEASE_FILL_IN
    client_id: PLEASE_FILL_IN
    client_secret: PLEASE_FILL_IN
    cloud_region: us-1
  registry:
    type: gcr
```

`cid` parameter refers to CrowdStrike Customer ID. This CID will be used to start Falcon Container sensors and all the data will be reported to that CID. `client_id` and `client_secret` parameters refer to API Key pairs used to download the CrowdStrike Falcon Container sensor (no other permission except the sensor download shall be granted to this API key pair).

When FalconContainer Resources is pushed to the cluster, falcon-operator will automatically install Falcon Container product to the cluster.

### Uninstall Steps
 - To uninstall Falcon Container simply remove FalconContainer resource. The operator will uninstall Falcon Container product from the cluster.

   ```
   kubectl delete falconcontainers.falcon.crowdstrike.com default
   ```
 - To uninstall Falcon Operator run
   ```
   operator-sdk cleanup falcon-operator --namespace falcon-operator
   ```

### Upgrades

Current version of the operator does not automatically updates Falcon Container sensor. Users are advised to remove & re-add FalconContainer resource to uninstall Falcon Container and to install the newest version.

### Namespace Reference

The following namespaces will be used by Falcon Operator.

| Namespace               | Description                                                      |
|:------------------------|:-----------------------------------------------------------------|
| falcon-system           | Used by Falcon Container product, runs the injector and webhoook |
| falcon-operator         | Runs falcon-operator manager                                     |
| falcon-system-configure | Used by operator, contains objects created by operator           |

### Compatibility Guide

Falcon Operator initially supports only GKE/GCR.

| Platform                | Supported versions                                     |
|:------------------------|:-------------------------------------------------------|
| GKE                     | 1.18, 1.19, 1.20                                       |

### Troubleshooting

Falcon Operator modifies the FalconContainer CRD based on what is happening in the cluster. Should an error occur during Falcon Container deployment that error will appear in kubectl output as shown below.

```
$ kubectl get falconcontainers.falcon.crowdstrike.com
NAME       STATUS   ERROR
default    DONE
```

The empty ERROR column together with status=DONE indicates that Falcon Container deployment did not yield any errors. Should more insight be needed, users are advised to view FalconContainer CRD in full detail

```
kubectl get falconcontainers.falcon.crowdstrike.com -o yaml
```

or to review the logs of Falcon Operator
```
kubectl -n falcon-operator logs -f deploy/falcon-operator-controller-manager -c manager
```

### Additional Documentation

 - [Deployment Guide for GKE](docs/deployment/gke/README.md)
 - [Developer Documentation](docs/developer_guide.md)

## Getting Help
If you encounter any issues while using Falcon Operator, you can create an issue on our [Github repo](https://github.com/CrowdStrike/falcon-operator) for bugs, enhancements, or other requests.

## Contributing
You can contribute by:

* Raising any issues you find using Falcon Operator
* Fixing issues by opening [Pull Requests](https://github.com/CrowdStrike/falcon-operator/pulls)
* Submitting a patch or opening a PR
* Improving documentation
* Talking about 3scale Operator

All bugs, tasks or enhancements are tracked as [GitHub issues](https://github.com/CrowdStrike/falcon-operator/issues).

## Additional Resources
 - CrowdStrike Container Security: [Product Page](https://www.crowdstrike.com/products/cloud-security/falcon-cloud-workload-protection/container-security/)
 - So You Think Your Containers Are Secure? Four Steps to Ensure a Secure Container Deployment: [Blog Post](https://www.crowdstrike.com/blog/four-steps-to-ensure-a-secure-containter-deployment/)
 - Container Security With CrowdStrike: [Blog Post](https://www.crowdstrike.com/blog/tech-center/container-security/)
 - To learn more about Falcon Container Sensor for Linux: [Deployment Guide](https://falcon.crowdstrike.com/support/documentation/146/falcon-container-sensor-for-linux), [Release Notes](https://falcon.crowdstrike.com/support/news/release-notes-falcon-container-sensor-for-linux)
