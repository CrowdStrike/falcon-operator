<!--- NOTE: DO NOT EDIT! This file is auto-generated. Please update the source *.tmpl file instead --->
# Deployment Guide for EKS Fargate and ECR
This document will guide you through the installation of the Falcon Operator and deployment of the following custom resources provided by the Falcon Operator:
- [FalconAdmission](../../resources/admission/README.md) with the Falcon Admission Controller image being mirrored from CrowdStrike container registry to ECR (Elastic Container Registry). A new AWS IAM Policy will be created to allow the operator to push to ECR registry.
- [FalconContainer](../../resources/container/README.md) with the Falcon Container image being mirrored from CrowdStrike container registry to ECR (Elastic Container Registry). A new AWS IAM Policy will be created to allow the operator to push to ECR registry.
- [FalconImageAnalyzer](../../resources/imageanalyzer/README.md) with the Falcon Image Analyzer image being pull from the CrowdStrike container registry.

## Prerequisites

> [!IMPORTANT]
> - The correct CrowdStrike Cloud (not Endpoint) subscription
> - CrowdStrike API Key Pair (*if installing the CrowdStrike Sensor via the CrowdStrike API*)
>
>    > If you need help creating a new API key pair, review our docs: [CrowdStrike Falcon](https://falcon.crowdstrike.com/support/api-clients-and-keys).
>
>  Make sure to assign the following permissions to the key pair:
>  - Falcon Images Download: **Read**
>  - Sensor Download: **Read**

## Installing the Falcon Operator

<details>
  <summary>Click to expand</summary>

- Set up a new Kubernetes cluster or use an existing one.
- Create an EKS Fargate profile for the operator:
  ```sh
  eksctl create fargateprofile \
    --region "$AWS_REGION" \
    --cluster eks-fargate-cluster \
    --name fp-falcon-operator \
    --namespace falcon-operator
  ```

- Install the Falcon Operator by running the following command:
  ```sh
  kubectl apply -f https://github.com/crowdstrike/falcon-operator/releases/latest/download/falcon-operator.yaml
  ```

</details>

### Deploying the Falcon Container Sidecar Sensor

<details>
  <summary>Click to expand</summary>

#### Create the FalconContainer resource

> [!IMPORTANT]
> If running in a mixed environment with both Fargate and EKS instances, you must set the installNamespace to a different namespace in the FalconContainer Spec i.e. `spec.installNamespace: falcon-Sidecar` to avoid conflicts with FalconNodeSensor running in the `falcon-system` namespace.

- Create an EKS Fargate profile for the FalconContainer resource deployment:
  ```sh
  eksctl create fargateprofile \
    --region "$AWS_REGION" \
    --cluster eks-fargate-cluster \
    --name fp-falcon-system \
    --namespace falcon-system
  ```


- Create a new FalconContainer resource
  ```sh
  kubectl create -f https://raw.githubusercontent.com/crowdstrike/falcon-operator/main/docs/deployment/eks-fargate/falconcontainer.yaml --edit=true
  ```



</details>

### Deploying the Falcon Admission Controller

<details>
  <summary>Click to expand</summary>

- Create an EKS Fargate profile for the FalconAdmission resource deployment:
  ```sh
  eksctl create fargateprofile \
    --region "$AWS_REGION" \
    --cluster eks-fargate-cluster \
    --name fp-falcon-kac \
    --namespace falcon-kac
  ```


- Create a new FalconAdmission resource
  ```sh
  kubectl create -f https://raw.githubusercontent.com/crowdstrike/falcon-operator/main/docs/deployment/eks-fargate/falconadmission.yaml --edit=true
  ```

</details>

### Deploying the Falcon Image Analyzer

<details>
  <summary>Click to expand</summary>

- Create an EKS Fargate profile for the FalconImageAnalyzer resource deployment:
  ```sh
  eksctl create fargateprofile \
    --region "$AWS_REGION" \
    --cluster eks-fargate-cluster \
    --name fp-falcon-iar \
    --namespace falcon-iar
  ```


After the Falcon Operator has deployed, you can now deploy the Image Analyzer:

- Deploy FalconImageAnalyzer through the cli using the `kubectl` command:
  ```sh
  kubectl create -n falcon-operator -f https://raw.githubusercontent.com/crowdstrike/falcon-operator/main/config/samples/falcon_v1alpha1_falconimageanalyzer.yaml --edit=true
  ```

</details>

## Upgrading

<details>
  <summary>Click to expand</summary>

To upgrade, run the following command:

```sh
kubectl apply -f https://github.com/crowdstrike/falcon-operator/releases/latest/download/falcon-operator.yaml
```

If you want to upgrade to a specific version, replace `latest` with the desired version number in the URL:

```sh
VERSION=1.2.3
kubectl apply -f https://github.com/CrowdStrike/falcon-operator/releases/download/${VERSION}/falcon-operator.yaml
```

</details>

## Uninstalling

> [!WARNING]
> It is essential to uninstall ALL of the deployed custom resources before uninstalling the Falcon Operator to ensure proper cleanup.

</details>

### Uninstalling the Falcon Container Sidecar Sensor

<details>
  <summary>Click to expand</summary>

Remove the FalconContainer resource. The operator will then uninstall the Falcon Container Sidecar Sensor from the cluster:

```sh
kubectl delete falconcontainers --all
```

</details>

### Uninstalling the Falcon Admission Controller

<details>
  <summary>Click to expand</summary>

Remove the FalconAdmission resource. The operator will then uninstall the Falcon Admission Controller from the cluster:

```sh
kubectl delete falconadmission --all
```

</details>

### Uninstalling the Falcon Image Analyzer

<details>
  <summary>Click to expand</summary>

Remove the FalconImageAnalyzer resource. The operator will then uninstall the Falcon Image Analyzer from the cluster:

```sh
kubectl delete falconimageanalyzer --all
```

</details>

### Uninstalling the Falcon Operator

<details>
  <summary>Click to expand</summary>

Delete the Falcon Operator deployment by running:

```sh
kubectl delete -f https://github.com/crowdstrike/falcon-operator/releases/latest/download/falcon-operator.yaml
```

</details>

## Configuring IAM Role to allow ECR Access on EKS Fargate

### Configure IAM Role for ECR Access for the Sidecar Injector

<details>
  <summary>Click to expand</summary>

When the Falcon Container Injector is installed on EKS Fargate, the following error message may appear in the injector logs:

```
level=error msg="Failed to handle webhook request" error="Failed to retrieve image details for \"123456789.dkr.ecr.region.amazonaws.com/deployment.example.com:latest\" in container \"app\" in pod \"default/\": Failed to get the image config/digest": error reading manifest latest: unauthorized: authentication required"
```

This may be an indication of the injector running with insufficient ECR privileges. This can happen
when the IAM role of the Fargate nodes is not propagated to the pods.

Conceptually, the following tasks need to be done in order to enable ECR pull from the injector:

- Create IAM Policy for ECR image pull
- Create IAM Role for the injector
- Assign the IAM Role to the injector (and set-up a proper trust relationship on the role and OIDC identity provider)
- Put IAM Role ARN into your Falcon Container resource for re-deployments

#### Assigning AWS IAM Role to Falcon Container Injector

<details>
  <summary>Click to expand</summary>

Using `aws`, `eksctl`, and `kubectl` command-line tools, perform the following steps:

- Set up your shell environment variables
  ```sh
  export AWS_REGION="insert your region"
  export EKS_CLUSTER_NAME="insert your cluster name"

  export AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
  iam_policy_name="FalconContainerEcrPull"
  iam_policy_arn="arn:aws:iam::${AWS_ACCOUNT_ID}:policy/${iam_policy_name}"
  ```

- Create AWS IAM Policy for ECR image pulling
  ```sh
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
  ```sh
  eksctl create iamserviceaccount \
         --name falcon-operator-sidecar-sensor \
         --namespace falcon-system \
         --region "$AWS_REGION" \
         --cluster "${EKS_CLUSTER_NAME}" \
         --attach-policy-arn "${iam_policy_arn}" \
         --approve \
         --override-existing-serviceaccounts
  ```

- Verify that the IAM Role (not to be confused with IAM Policy) has been assigned to the ServiceAccount by the previous command:
  ```sh
  kubectl get sa -n falcon-system falcon-operator-sidecar-sensor -o=jsonpath='{.metadata.annotations.eks\.amazonaws\.com/role-arn}'
  ```

- Delete the previously deployed FalconContainer resource:
  ```sh
  kubectl delete falconcontainers --all
  ```

- Add Role ARN to your FalconContainer yaml file:
  ```yaml
    injector:
      serviceAccount:
        annotations:
          eks.amazonaws.com/role-arn: arn:aws:iam::12345678910:role/eksctl-demo-cluster-addon-iamservic-Role1-J78KUNY32R1
  ```

- Deploy the FalconContainer resource with the IAM role changes:
  ```sh
  kubectl create -f ./my-falcon-container.yaml
  ```
  
</details>
</details>

### Configure IAM Role for ECR Access for the Admission Controller

<details>
  <summary>Click to expand</summary>

When the Falcon Admission Controller is installed on EKS Fargate, you may need to enable ECR access for the admission controller. 
Conceptually, the following tasks need to be done in order to enable ECR pull from the admission controller:

- Create IAM Policy for ECR image pull
- Create IAM Role for the admission controller
- Assign the IAM Role to the admission controller (and set-up a proper trust relationship on the role and OIDC identity provider)
- Put IAM Role ARN into your Falcon Admission resource for re-deployments

#### Assigning AWS IAM Role to Falcon Admission Controller

<details>
  <summary>Click to expand</summary>

Using `aws`, `eksctl`, and `kubectl` command-line tools, perform the following steps:

- Set up your shell environment variables
  ```sh
  export AWS_REGION="insert your region"
  export EKS_CLUSTER_NAME="insert your cluster name"

  export AWS_ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
  iam_policy_name="FalconAdmissionEcrPull"
  iam_policy_arn="arn:aws:iam::${AWS_ACCOUNT_ID}:policy/${iam_policy_name}"
  ```

- Create AWS IAM Policy for ECR image pulling
  ```sh
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
      --description "Policy to enable Falcon Admission Controller to pull container image from ECR"
  ```

- Assign the newly created policy to the kubernetes ServiceAccount of Falcon Admission Controller
  ```sh
  eksctl create iamserviceaccount \
         --name falcon-operator-admission-controller \
         --namespace falcon-kac \
         --region "$AWS_REGION" \
         --cluster "${EKS_CLUSTER_NAME}" \
         --attach-policy-arn "${iam_policy_arn}" \
         --approve \
         --override-existing-serviceaccounts
  ```

- Verify that the IAM Role (not to be confused with IAM Policy) has been assigned to the ServiceAccount by the previous command:
  ```sh
  kubectl get sa -n falcon-kac falcon-operator-admission-controller -o=jsonpath='{.metadata.annotations.eks\.amazonaws\.com/role-arn}'
  ```

- Delete the previously deployed FalconAdmission resource:
  ```sh
  kubectl delete falconadmission --all
  ```

- Add Role ARN to your FalconAdmission yaml file:
  ```yaml
    admissionConfig:
      serviceAccount:
        annotations:
          eks.amazonaws.com/role-arn: arn:aws:iam::12345678910:role/eksctl-demo-cluster-addon-iamservic-Role1-J78KUNY32R1
  ```

- Deploy the FalconAdmission resource with the IAM role changes:
  ```sh
  kubectl create -f ./my-falcon-admission.yaml
  ```
  
</details>
</details>
