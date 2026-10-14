# Installing with `oc mirror`

> [!IMPORTANT]
> The Falcon sensor and other Falcon Kubernetes components require a connection the CrowdStrike Cloud. These components do not support fully air-gapped clusters. Even if your OpenShift cluster is deployed with `oc mirror`, the cluster must support outbound connections to the CrowdStrike Cloud, e.g. through a proxy.

This guide will only cover the requirements for using `oc mirror` with the Falcon operator. For the full lifecycle of deploying a cluster with `oc mirror`, refer to the [Disconnected environments](https://docs.redhat.com/en/documentation/openshift_container_platform/4.20/html/disconnected_environments/index) section of the OpenShift documentation. The guide below was specifically written to go along with [Mirroring images for a disconnected installation by using the oc-mirror plugin v2](https://docs.redhat.com/en/documentation/openshift_container_platform/4.20/html/disconnected_environments/about-installing-oc-mirror-v2).

## Summary

During the course of a cluster deployment with `oc mirror`, there are three important steps to enable the Falcon operator to be mirrored:

1. Ensure your pull secret file contains credentials for the CrowdStrike registry.
2. Configure your `ImageSetConfiguration` to mirror region-specific container images.
3. Upon sensor deployment, reference your mirrored images.

> [!NOTE]
> Please keep in mind this guide is maintained as a best effort by CrowdStrike solution architects. CrowdStrike engineering does not formally test on a mirror-installed OpenShift cluster. If you experience any trouble, talk to your cloud sales engineer or file an issue on this repository.

## Gather CrowdStrike Pull Secret

During the course of the mirroring process, you will prepare a pull secrets file containing your Red Hat pull secret and credentials for your mirror registry. Because CrowdStrike container images require authentication, you also need to add credentials for the CrowdStrike registry to your pull secret.

**Prerequisite:** Prepare your pull secret file as described in the OpenShift documentation, containing your Red Hat pull secret and mirror registry credentials.

1. [Obtain the falcon-container-sensor-pull script](https://github.com/CrowdStrike/falcon-scripts/tree/main/bash/containers/falcon-container-sensor-pull) and follow all installation and setup requirements
2. Ensure `FALCON_CLIENT_ID` and `FALCON_CLIENT_SECRET` are present in your environment (see script documentation above for more details)
3. Get and decode the CrowdStrike pull token:

```bash
$ ./falcon-container-sensor-pull.sh -t falcon-sensor --get-pull-token | base64 -d
{"auths": { "registry.crowdstrike.com": { "auth": "aBc123...XyZ789" } } }
```

4. Copy the contents of the `auths` section into your pull secret file, ensuring you add a comma to the list where necessary:

```json
{
  "auths": {
    "cloud.openshift.com": {
      "auth": "b3BlbnNo...",
      "email": "you@example.com"
    },
    "quay.io": {
      "auth": "b3BlbnNo...",
      "email": "you@example.com"
    },
    "registry.connect.redhat.com": {
      "auth": "NTE3Njg5Nj...",
      "email": "you@example.com"
    },
    "registry.redhat.io": {
      "auth": "NTE3Njg5Nj...",
      "email": "you@example.com"
    },
    "registry.crowdstrike.com": {
      "auth": "aBc123...XyZ789",
    }
  }
}
```

## Update `ImageSetConfiguration` File

CrowdStrike sensors, and therefore container images, are historically tied to the region your Falcon CID (tenant) is deployed into (e.g. us-1, eu-1). The Falcon operator contains references to us-1 only for publishing purposes. You need to instruct `oc mirror` to ignore these default references, and instead pull your region-specific images instead.

**Prerequisite:** Prepare your `ImageSetConfiguration` file as necessary for your environment. The example below is just a reference for where to make changes specific to this operator. It will not match your environment's specific configuration.

1. Add the Falcon operator to `operators`
1. Add the default us-1 images to `blockedImages`
1. Add your desired images to `additionalImages`

```yaml
kind: ImageSetConfiguration
apiVersion: mirror.openshift.io/v2alpha1
mirror:
  operators:
    - catalog: registry.redhat.io/redhat/certified-operator-index:v4.22
      packages:
      - name: falcon-operator
        channels:
        - name: certified-1.0
          minVersion: 1.15.0
  blockedImages:
    - name: registry.crowdstrike.com/falcon-container/us-1
    - name: registry.crowdstrike.com/falcon-imageanalyzer/us-1
    - name: registry.crowdstrike.com/falcon-sensor/us-1
    - name: registry.crowdstrike.com/falcon-kac/us-1
  additionalImages:
    - name: registry.crowdstrike.com/falcon-sensor/release/falcon-sensor:8.10.0-19403-1
    - name: registry.crowdstrike.com/falcon-kac/release/falcon-kac:8.11.0-4003
    - name: registry.crowdstrike.com/falcon-imageanalyzer/release/falcon-imageanalyzer:1.0.27
```

## Use Mirrored Images

When deploying the Falcon CRD's, ensure you are specifying the images in your mirror registry:

| CRD                   | YAML path to specify image  |
| --------------------- | --------------------------- |
| `FalconNodeSensor`    | `spec.node.image`           |
| `FalconAdmission`     | `spec.image`                |
| `FalconImageAnalyzer` | `spec.image`                |
