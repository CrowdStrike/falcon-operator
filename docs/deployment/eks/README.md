# Deployment Guide for EKS & ECR

## Pre-requisites

 - Have CrowdStrike CWP subscription with Falcon Container enabled (have your CrowdStrike CID ready)
 - Create new CrowdStrike API key pair with permission to download the sensor (no other permission shall be required)

## EKS/ECR Environment Preparation

EKS cluster that runs Falcon Operator needs to have [IAM OIDC provider](https://docs.aws.amazon.com/eks/latest/userguide/iam-roles-for-service-accounts.html) installed. The IAM OIDC provider associates AWS IAM roles with EKS workloads. Please review [AWS documentation](https://docs.aws.amazon.com/eks/latest/userguide/enable-iam-roles-for-service-accounts.html) to understand how the IAM OIDC provider works before proceeding. The script bellow will install IAM OIDC provider on your cluster.

 - Provide  settings as environment variables
   ```
   export AWS_REGION=
   export EKS_CLUSTER_NAME=
   ```
 - To install IAM OIDC on the cluster:
   ```
   eksctl utils associate-iam-oidc-provider --region "$AWS_REGION" --cluster "$EKS_CLUSTER_NAME" --approve
   ```

## Installation Steps

 - Open AWS Cloud Shell: https://console.aws.amazon.com/cloudshell/home

 - Install the operator & deploy Falcon Container Sensor
   ```
   bash -c 'source <(curl -s https://raw.githubusercontent.com/CrowdStrike/falcon-operator/main/docs/deployment/eks/run)'
   ```
   Note this script should be run as in the cloud shell as it will attempt to install kubectl, eksctl and operator-sdk command-line tools if needed.

## Uninstall Steps

 - To uninstall Falcon Container simply remove FalconContainer resource. The operator will uninstall Falcon Container product from the cluster.
   ```
   kubectl delete falconcontainers.falcon.crowdstrike.com default
   ```
 - To uninstall Falcon Operator that was installed using Operator Lifecycle manager
   ```
   operator-sdk cleanup falcon-operator --namespace falcon-operator
   ```
 - To uninstall Falcon Operator that was installed without Operator Lifecycle manager
   ```
   kubectl delete -f https://raw.githubusercontent.com/CrowdStrike/falcon-operator/main/deploy/falcon-operator.yaml
   ```
