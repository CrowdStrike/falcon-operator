# Installation and Deployment Guide

## Installation and Deployment

The Falcon Operator offers various installation and deployment options for specific Kubernetes distributions. The guides below provide detailed instructions for each case.

> [!WARNING]
> If none of the guides provide installation for your specific Kubernetes distribution, use the [Deployment Guide for Generic Kubernetes](./deployment/generic/README.md).

### Kubernetes Distribution Installation and Deployment

For an optimal experience, use the following preferred methods when installing for specific Kubernetes Distributions:

- **[Deployment Guide for AKS/ACR](./deployment/azure/README.md)**
- **[Deployment Guide for EKS/ECR](./deployment/eks/README.md)**
- **[Deployment Guide for EKS Fargate](./deployment/eks-fargate/README.md)**
- **[Deployment Guide for GKE/GCR](./deployment/gke/README.md)**
- **[Deployment Guide for OpenShift](./deployment/openshift/README.md)**
- **[Deployment Guide for Generic Kubernetes](./deployment/generic/README.md)**

## Upgrading

Falcon Operator and Sensor management and upgrades are best handled by using GitOps methodologies and workflows. Multi-Cluster Management tools such as [Red Hat Advanced Cluster Management for Kubernetes](https://www.redhat.com/en/technologies/management/advanced-cluster-management) or [SuSE Rancher](https://www.rancher.com/products/rancher) can help when needing to scale management across multiple clusters from Git workflows. Using GitOps ensures several best operational and security practices around Kubernetes as it is the configuration management tool of Kubernetes:

1. Containers are immutable and are meant to be immutable. This means that a container should not be modified during its life: no updates, no patches, no configuration changes. Immutable containers ensures
   deployments are safe, consistently repeatable, and makes it easier to roll back an upgrade in case of a problem. If a container is modified or drifts from its original build, this could be an indication of an attack compromise.
2. Kubernetes expands on the concept of container immutability by creating and coalescing around the concept of Immutable Infrastructure: changes e.g. upgrades deploy a new version with no upgrade in place.
3. Latest versions of released components should always be used which means no more N-1, N-2, etc. for sensor deployments.
4. No upgrades should happen outside the configuration management tool.

### Operator Upgrades

[See the individual deployment guides for commands on how to upgrade the operator](#kubernetes-distribution-installation-and-deployment).

### Sensor Upgrades

To effectively deploy and use the Falcon sensor in a Kubernetes environment, the following is recommended for the reasons listed above:

1. Copy the CrowdStrike sensor(s) to your own container registry.
2. Use Git to store the [FalconNodeSensor](https://github.com/crowdstrike/falcon-operator/tree/main/config/samples) and/or [FalconContainer](https://github.com/crowdstrike/falcon-operator/tree/main/config/samples) Kind(s) specifying the sensor in your internal registry.
3. Alway use the latest sensor version as soon as it is released and able to be updated in your environment.
4. As soon as the sensor version is changed in Git, a CI/CD pipeline should update the FalconNodeSensor and/or FalconContainer Kind(s) which will then cause the operator to deploy the updated versions to your Kubernetes environments. This is the proper way to handle sensor updates in Kubernetes.
5. Upgrades should usually happen in a rolling update manner to ensure the Kubernetes cluster and deployed resources stay accessible and operational.

## FAQ - Frequently Asked Questions

### What network connections are required for the operator to work properly?

- The operator image is hosted at [quay.io/crowdstrike/falcon-operator](https://quay.io/crowdstrike/falcon-operator). If necessary, the operator image itself can be mirrored to your registry of choice, including internally hosted registries.
- The operator must access your specific Falcon cloud region (`api.crowdstrike.com` or `api.[YOUR CLOUD].crowdstrike.com`).
- Depending on whether the image is mirrored, the operator or your nodes may need access to `registry.crowdstrike.com`.
- If Falcon Cloud is set to autodiscover, the operator may also attempt to reach the Falcon Cloud Region **us-1**.
- If a proxy is configured, please ensure appropriate connections are allowed to Falcon Cloud; otherwise, the operator or custom resource may not deploy correctly.

## Troubleshooting

To review the logs of Falcon Operator:

```shell
kubectl -n falcon-operator logs -f deploy/falcon-operator-controller-manager -c manager
```

### Operator Issues

#### Resources stuck in PodInitializing state indefinitely

If a cluster-wide nodeSelector policy is in place, this must be disabled in the namespaces that the sensors are deployed.

For example, on OpenShift:

```shell
oc annotate ns falcon-operator openshift.io/node-selector=""
```

#### Falcon Operator Controller Manager - OOMKilled
If the Falcon Operator Controller Manager becomes OOMKilled on startup, it could be due to the number and size of resources in the Kubernetes cluster that it has to monitor.
The OOMKilled error looks like:

```shell
$ kubectl get pods -n falcon-operator
NAME                                                  READY   STATUS      RESTARTS      AGE
falcon-operator-controller-manager-77d7b44f96-t6jsr   1/2     OOMKilled   2 (45s ago)   98s
```

To remediate this problem in an OpenShift cluster, increase the memory limit of the operator by adding the desired resource configuration to the Subscription:

```shell
oc edit subscription falcon-operator -n falcon-operator
```

and add/edit the resource configuration to the `spec`. For example:

```yaml
spec:
  channel: certified-0.9
  config:
    resources:
      limits:
        cpu: 500m
        memory: 128Mi
      requests:
        cpu: 250m
        memory: 64Mi
```

#### Falcon Operator Controller Manager - Failed Lease Renewal
During times when host resources are limited or there is increased network latency, the Falcon Operator may have trouble renewing renewing its lease when Leader Election is enabled. An example of this error can be seen in the `falcon-operator-controller-manager` pod logs:
```
2024-07-06T06:27:38Z    INFO    Updating FalconAdmission Deployment     {"controller": "falconadmission", "controllerGroup": "falcon.crowdstrike.com", "controllerKind": "FalconAdmission", "FalconAdmission": {"name":"falcon-admission"}, "namespace": "", "name": "falcon-admission", "reconcileID": "b3767fff-6acf-4c8d-a191-2d2194eb3e72", "Deployment.Name": "falcon-admission", "Deployment.Namespace": "falcon-kac"}
E0706 06:39:08.445673       1 leaderelection.go:327] error retrieving resource lock falcon-operator/falcon-operator-lock: Get "https://172.20.0.1:443/apis/coordination.k8s.io/v1/namespaces/falcon-operator/leases/falcon-operator-lock": context deadline exceeded
I0706 06:39:08.445717       1 leaderelection.go:280] failed to renew lease falcon-operator/falcon-operator-lock: timed out waiting for the condition
2024-07-06T06:39:08Z    ERROR   setup   problem running manager {"error": "leader election lost"}
```

To remediate this problem, add the following flags under `Deployment.spec.template.spec.containers.[0].args` in the falcon-operator.yaml manifest located [here](../deploy/falcon-operator.yaml):<br>

```
---
apiVersion: apps/v1
kind: Deployment
spec:
  template:
    spec:
      containers:
      - args:
        - --leader-elect
        - --lease-duration=DURATION
        - --renew-deadline=DURATION
```
`--lease-duration=DURATION` - This is the maximum duration that a leader can be stopped before it is replaced by another candidate. <br>
`--renew-deadline=DURATION` - The interval between attempts by the acting master to renew a leadership slot before it stops leading. Must be smaller than `lease-duration`.<br>
