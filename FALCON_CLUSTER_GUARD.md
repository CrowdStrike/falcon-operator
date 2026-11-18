# Announcement: Prepare Your Kubernetes Environment for Migrating to Falcon Cluster Guard in January 2027

## Summary

The December 2026 releases of the Falcon sensor for Linux (DaemonSet), Kubernetes Admission Controller (KAC), and Image Assessment at Runtime (IAR) will be their final standalone versions.

In January 2027, CrowdStrike will consolidate those three features into Falcon Cluster Guard, a unified Kubernetes security suite that eliminates the need to manage three separate deployments.

## Applies To

- **Requires one or more of these subscriptions:**
  - Falcon Cloud Security with Containers CNAPP
  - Falcon Cloud Security with Containers Runtime Protection
  - Falcon Cloud Security with Containers Cloud Detection and Response
  - Falcon Cloud Security with Containers
- **Platform:** Kubernetes clusters running the Falcon sensor for Linux (DaemonSet), the Kubernetes Admission Controller, or Image Assessment at Runtime
  - Non-DaemonSet deployments and Container Sensor deployments are not affected
- **CrowdStrike clouds:** Available in all clouds

### Affected Falcon Operator Custom Resources

| Product                                   | Custom Resource                                                       | Affected |
|:------------------------------------------|:----------------------------------------------------------------------|:---------|
| Falcon sensor for Linux (DaemonSet)       | [FalconNodeSensor](docs/resources/node/README.md)                     | Yes      |
| Kubernetes Admission Controller (KAC)     | [FalconAdmission](docs/resources/admission/README.md)                 | Yes      |
| Image Assessment at Runtime (IAR)         | [FalconImageAnalyzer](docs/resources/imageanalyzer/README.md)         | Yes      |
| Falcon Container Sensor                   | [FalconContainer](docs/resources/container/README.md)                 | No       |
| All of the above from a single manifest   | [FalconDeployment](docs/resources/falcondeployment/README.md)         | Yes, if it deploys any affected component |

## Details

The Falcon sensor for Linux (DaemonSet), KAC, and IAR currently operate as separate components. Falcon Cluster Guard unifies them into a single solution, enabling continuous context sharing and intelligence across components. This provides a complete, correlated view of your cluster, from deployment posture to workload behavior.

## Action Required

To prepare for Falcon Cluster Guard, we recommend identifying any DaemonSet sensor, KAC, or IAR products installed in your environments, and the corresponding scripts or the CI/CD workflows you use to deploy them. Having this inventory early will simplify your transition to Falcon Cluster Guard.

### Identify Falcon Operator Deployments

To list the affected custom resources managed by the Falcon Operator on a cluster, run:

```sh
kubectl get falconnodesensors,falconadmissions,falconimageanalyzers,falcondeployments
```

### Identify Installations with Next-Gen SIEM

To identify these products in your environment via Next-Gen SIEM Advanced Event Search, use the below queries.

List out all KAC installations online within the last week:

```
setTimeInterval(start="7d")
| event_platform="K8s" #event_simpleName="K8SClusterInfo"
| groupBy([K8SClusterId, K8SClusterName], limit=max, function=[count(aid, distinct=true)])
```

List out all IAR installations online within the last week:

```
setTimeInterval(start="7d")
| #event_simpleName=K8SCluster aid=iar
| groupBy([K8SClusterId, K8SClusterName], limit=max, function=[count(K8SClusterId, distinct=true)])
```

List out all Kubernetes clusters with DaemonSet Linux sensors online within the last week:

```
setTimeInterval(start="7d")
| event_platform="Lin" #event_simpleName=AgentOnline K8SClusterId=*
| groupBy([K8SClusterId], limit=max, function=[count(K8SClusterId, distinct=true)])
```

### When Falcon Cluster Guard Launches

When Falcon Cluster Guard launches, a single unified image will be published to the CrowdStrike registry. Once available, please complete the following steps:

- Update your scripts and CI/CD pipelines to reference the new image.
- Upgrade your Helm and Operator deployments using the upgrade and configuration instructions that will be shared prior to launch.

CrowdStrike will provide additional updates and migration guidance before the migration date.

## Resources

- Read our [documentation](https://docs.crowdstrike.com/)
- See [SUPPORT.md](SUPPORT.md) for Falcon Operator support options
