# Deployment Guide for OpenShift

This document guides you through the recommended installation of the CrowdStrike Falcon agent on OpenShift, including self-managed OpenShift and OpenShift cloud services. This approach enables CrowdStrike's breach prevention on Red Hat Enterprise Linux CoreOS (the operating system that powers OpenShift), as well as all container workloads running on top of it. Control plane and worker nodes are both protected by default.

The Falcon agent is deployed as a certified operator in OpenShift's OperatorHub. This guide documents installation in three forms:

- Via the web console, for quick evaluations.
- Via the command line, for quick evaluations and script-based configuration management workflows.
- Via manifest files, for configuration-as-code and GitOps approaches to security. Recommended for production deployments to ensure consistency and reproducibility.

You only need to follow one installation method per section.

## Prerequisites

- A Falcon Cloud Security for Containers subscription (previously known as Cloud Workload Protection)
- Red Hat Openshift 4.10+ with `cluster-admin` privileges:
  - Self-managed OpenShift on any platform
  - Red Hat OpenShift Service on AWS (ROSA)
  - Azure Red Hat OpenShift (ARO)
  - Red Hat OpenShift on IBM Cloud (RHOIC)

## Limitations

This guide covers deployment of the FalconNodeSensor custom resource, which is the recommended deployment method for OpenShift because it provides protection for both CoreOS and all container workloads. In very rare circumstances where you cannot deploy the FalconNodeSensor, refer to the [FalconContainer deployment guide](README-container.md).

This guide also covers only the default installation options, which are sufficient for most deployments. For a list of all configuration options, see [FalconNodeSensor](resources/node/README.md) or [FalconContainer](resources/container/README.md).

## Overview

The deployment process consists of three steps:

1. In the Falcon platform, create a new API client with the "Falcon Images Download: Read" and "Sensor Download: Read" permissions. Note your client ID and secret.
2. In OpenShift, install the Falcon operator (use the Marketplace operator).
3. Once the Falcon operator is installed, create a FalconNodeSensor, and provide your new API client’s ID and secret.

The Falcon agent will be deployed to all nodes in the cluster and immediately start providing protection, no reboots or redeploys needed. The following sections provide details on each of these steps.

## Create a Falcon API client

To discover your customer ID and download the sensor image, the operator will connect to the Falcon API. You'll need to provide the API client ID and secret to the operator.

1. Navigate to _API clients and keys_ ([US-1](https://falcon.crowdstrike.com/api-clients-and-keys/clients), [US-2](https://falcon.us-2.crowdstrike.com/api-clients-and-keys/clients)).
1. Click `Create API client`.
1. Provide a name and description and the following permissions:
   - Falcon Images Download: Read
   - Sensor Download: Read
1. Click `Create`.
1. Note the client ID and secret, you'll need it in the following steps.

## Install the Falcon operator

### Option 1: Via the web console

- Log in to your OpenShift cluster

   ![OpenShift Web Console Login](images/ocp-login.png)

- Click on the `Operators` dropdown. Then, click on `OperatorHub`

   ![OpenShift OperatorHub](images/ocp-ophub.png)

- Enter `crowdstrike` into the search bar, and click on the `CrowdStrike Falcon Platform - Operator` tile.

   ![OpenShift Search](images/ocp-optile.png)

- In the side menu, click the `Install` button.

   ![OpenShift CrowdStrike Operator Install](images/ocp-opinstall.png)

- Make any necessary changes as desired to the `InstallPlan` before installing the operator. You can set the update approval to `Automatic` which is the default or `Manual`. If you set to `Manual`, updates require approval before an operator will update.
  You can also set the desired update channel for OpenShift to check for updates. Please note that installation versions are tied to channels, and versions may not exist in every channel. Click the `Install` button to begin the install.

   ![OpenShift CrowdStrike Operator Install](images/ocp-opinstall2.png)

- Once the operator has completed installation, you can now deploy the custom resources the operator provides.

   ![OpenShift CrowdStrike Operator](images/ocp-opresources.png)

### Option 2: Via the CLI

The operator is easily installed using Krew and its operator management plugin:

1. Install Krew. See https://krew.sigs.k8s.io/docs/user-guide/setup/install/
2. Verify install with `oc krew`
3. Update krew `oc krew update`
4. Install the operator krew plugin `oc krew install operator`

Once the Krew plugin is installed:

- Log in to your OpenShift cluster
  ```
  oc login --token=sha256~abcde-ABCDE-1 --server=https://openshift.example.com
  ```

- Create the `falcon-operator` namespace:
  ```
  oc new-project falcon-operator
  ```

- Using the krew plugin, install the certified operator
   ```
   oc operator install falcon-operator-rhmp --create-operator-group -n falcon-operator
   ```

### Option 3: Via manifest files

Installing the operator via manifest files allows you to check these files into Git and use a configuration-as-code or GitOps approach to security management.

- Log in to your OpenShift cluster
  ```
  oc login --token=sha256~abcde-ABCDE-1 --server=https://openshift.example.com
  ```

- Create the `falcon-operator` namespace:
  ```
  oc new-project falcon-operator
  ```

- Verify that the Falcon Operator exists in the cluster's OperatorHub
  ```
  oc get packagemanifests -n openshift-marketplace | grep falcon
  # falcon-operator                                    Community Operators   18h
  # falcon-operator-rhmp                               Red Hat Marketplace   18h
  ```

- Create an `OperatorGroup` to allow the operator to be installed in the `falcon-operator` namespace (you can [review operatorgroup.yaml](operatorgroup.yaml)):
  ```
  oc create -f https://raw.githubusercontent.com/CrowdStrike/falcon-operator/main/docs/deployment/openshift/operatorgroup.yaml
  ```

- Create a `Subscription` to install the operator (you can [review redhat-subscription.yaml](redhat-subscription.yaml)):
  ```
  oc create -f https://raw.githubusercontent.com/CrowdStrike/falcon-operator/main/docs/deployment/openshift/redhat-subscription.yaml
  ```

## Deploy the sensor

### Option 1: Via the web console

- To deploy the Falcon Node Sensor, click `Create instance` for the `Falcon Node Sensor` Kind under the `Provided APIs` for the Falcon Operator.

   ![OpenShift CrowdStrike Falcon Node Sensor](images/ocp-fns.png)

- Enter your API client ID and secret under `Falcon Platform API Configuration`, then click `Create`.

   ![OpenShift CrowdStrike Falcon Node Sensor](images/ocp-fnsinstall.png)

### Option 2: Via the CLI

Deploying the Falcon sensors via the CLI and with manifest files are the same process. See the next section, _Option 3: Via manifest files_.

### Option 3: Via manifest files

Once the operator has deployed, you can now deploy the FalconNodeSensor.

- Deploy FalconNodeSensor using the `oc` command, supplying your API client ID and secret in `spec.falcon_api.client_id` and `client_secret`:
  ```
  oc create -n falcon-operator -f https://raw.githubusercontent.com/CrowdStrike/falcon-operator/main/config/samples/falcon_v1alpha1_falconnodesensor.yaml --edit=true
  ```
