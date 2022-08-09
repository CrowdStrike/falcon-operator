# Deployment Guide for EKS Fargate and ECR
This document will guide you through the installation of falcon-operator and deployment of either the:
- [FalconContainer](../../cluster_resources/container/README.md) custom resource to the cluster with Falcon Container image being mirrored from CrowdStrike container registry to ECR (Elastic Container Registry). A new AWS IAM Policy will be created to allow the opeator to push to ECR registry.

## Prerequisites

- CrowdStrike CWP subscription
- If your are installing the CrowdStrike Sensor via the Crowdstrike API, you need to create a new CrowdStrike API key pair with the following permissions:
  - Falcon Images Download: Read
  - Sensor Download: Read

## Installing the operator

- Create an EKS Fargate profile for the operator:
  ```
  eksctl create fargateprofile \
    --region "$AWS_REGION" \
    --cluster eks-fargate-cluster \
    --name fp-falcon-operator \
    --namespace falcon-operator
  ```
  
- Install the operator
  ```
  kubectl apply -f https://raw.githubusercontent.com/CrowdStrike/falcon-operator/main/deploy/falcon-operator.yaml
  ```

### Deploy the sidecar sensor
#### Create the FalconContainer resource

- Create an EKS Fargate profile for the FalconContainer resource deployment:
  ```
  eksctl create fargateprofile \
    --region "$AWS_REGION" \
    --cluster eks-fargate-cluster \
    --name fp-falcon-system \
    --namespace falcon-system
  ```

- Create an EKS Fargate profile for the FalconContainer resource deployment Kubernetes Job:
  ```
  eksctl create fargateprofile \
    --region "$AWS_REGION" \
    --cluster eks-fargate-cluster \
    --name fp-falcon-system-configure \
    --namespace falcon-system-configure
  ```

- Create a new FalconContainer resource
  ```
  kubectl create -f https://raw.githubusercontent.com/CrowdStrike/falcon-operator/main/docs/deployment/eks/falconcontainer.yaml --edit=true
  ```
  
## Uninstalling

When uninstalling the operator, it is important to make sure to uninstall the deployed custom resources first *before* you uninstall the operator.
This will insure proper cleanup of the resources.

### Uninstall the Sidecar Sensor

- To uninstall Falcon Container, simply remove the FalconContainer resource. The operator will then uninstall the Falcon Container product from the cluster.
  ```
  kubectl delete falconcontainers --all
  ```

### Uninstall the Operator

- To uninstall Falcon Operator, delete the deployment:
  ```
  kubectl delete -f https://raw.githubusercontent.com/CrowdStrike/falcon-operator/main/deploy/falcon-operator.yaml
  ```
  
## Configuring IAM Role to allow ECR Access on EKS Fargate

When the Falcon Container Injector is installed on EKS Fargate, the following error message may appear in the injector logs:

```
level=error msg="Failed to handle webhook request" error="Failed to retrieve image details for \"123456789.dkr.ecr.region.amazonaws.com/deployment.example.com:latest\" in container \"app\" in pod \"default/\": Failed to get the image config/digest": error reading manifest latest: unauthorized: authentication required"
```

This may be an indication of the injector running with insufficient ECR privileges. This can happen
when the IAM role of the Fargate nodes is not propagated to the pods.

Conceptually, the following tasks need to be done in order to enable ECR pull from the injector:

- Create IAM Policy for ECR image pull
- Create IAM Role for the injector
- Assign the IAM Role to the injector (and set-up a proper trust relationship on the role and OIDC indentity provider)
- Put IAM Role ARN into your Falcon Container resource for re-deployments

### Assigning AWS IAM Role to Falcon Container Injector

Using `aws`, `eksctl`, and `kubectl` command-line tools, perform the following steps:

- Set up your shell environment variables
  ```
  export AWS_REGION="insert your region"
  export EKS_CLUSTER_NAME="insert your cluster name"

  export AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
  iam_policy_name="FalconContainerEcrPull"
  iam_policy_arn="arn:aws:iam::${AWS_ACCOUNT_ID}:policy/${iam_policy_name}"
  ```

- Create AWS IAM Policy for ECR image pulling
  ```
  cat <<__END__ > policy.json
  {
      "Version": "2012-10-17",
      "Statement": [
          {
              "Sid": "AllowImagePull",
              "Effect": "Allow",
              "Action": [
                  "ecr:BatchGetImage",
                  "ecr:DescribeImages",
                  "ecr:GetDownloadUrlForLayer",
                  "ecr:ListImages"
              ],
              "Resource": "*"
          },
          {
              "Sid": "AllowECRSetup",
              "Effect": "Allow",
              "Action": [
                  "ecr:GetAuthorizationToken"
              ],
              "Resource": "*"
          }
      ]
  }
  __END__

  aws iam create-policy \
      --region "$AWS_REGION" \
      --policy-name ${iam_policy_name} \
      --policy-document 'file://policy.json' \
      --description "Policy to enable Falcon Container Injector to pull container image from ECR"
  ```

- Assign the newly created policy to the kubernetes ServiceAccount of Falcon Container Injector
  ```
  eksctl create iamserviceaccount \
         --name default \
         --namespace falcon-system \
         --region "$AWS_REGION" \
         --cluster "${EKS_CLUSTER_NAME}" \
         --attach-policy-arn "${iam_policy_arn}" \
         --approve \
         --override-existing-serviceaccounts
  ```

- Verify that the IAM Role (not to be confused with IAM Policy) has been assigned to the ServiceAccount by the previous command:
  ```
  kubectl get sa -n falcon-system default -o=jsonpath='{.metadata.annotations.eks\.amazonaws\.com/role-arn}'
  ```

- Delete the previously deployed FalconContainer resource:
  ```
  kubectl delete falconcontainers --all
  ```

- Add Role ARN to your FalconContainer yaml file:
  ```
    injector:
      sa_annotations:
        eks.amazonaws.com/role-arn: arn:aws:iam::12345678910:role/eksctl-demo-cluster-addon-iamservic-Role1-J78KUNY32R1
  ```

- Deploy the FalconContainer resource with the IAM role changes:
  ```
  kubectl create -f ./my-falcon-container.yaml
  ```