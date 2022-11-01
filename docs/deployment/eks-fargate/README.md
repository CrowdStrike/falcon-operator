# Considerations of using AWS ECR on EKS Fargate

When Falcon Container Injector [is installed on AWS EKS](../eks) the following error message may appear in the injector logs:

```
level=error msg="Failed to handle webhook request" error="Failed to retrieve image details for \"123456789.dkr.ecr.region.amazonaws.com/deployment.example.com:latest\" in container \"app\" in pod \"default/\": Failed to get the image config/digest": error reading manifest latest: unauthorized: authentication required"
```

This is may be an indication of the injector running with insufficient ECR privileged. That usually happens on EKS Fargate,
when IAM role of Fargate nodes is not propagated to the pods. The following document describes remediation steps.


## Assigning AWS IAM Role to Falcon Container Injector

Conceptually, the following tasks need to be done in order to enable ECR pull from the injector

 - Create IAM Policy for ECR image pull
 - Create IAM Role for the injector
 - Assign the IAM Role to the injector (and set-up a proper trust relationship on the role and OIDC indentity provider)
 - Put IAM Role ARN into your Falcon Container resource for re-deployments

The following step-by-step guide uses `aws`, `eksctl` and `kubectl` command-line tools to achieve that.

## Step-by-step guide to add ECR pull permission to the injector

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
- Display IAM Role (not to be confused with IAM Policy) that has been assigned to the ServiceAccount by the previous command
  ```
  kubectl get sa -n falcon-system default -o=jsonpath='{.metadata.annotations.eks\.amazonaws\.com/role-arn}'
  ```
 
- Re-deploy (delete & create) FalconContainer with the above IAM Role in the spec:
  ```
  kubectl delete falconcontainers --all
  ```

  Add Role ARN to your FalconContainer yaml file:
  ```
    injector:
      serviceAccount:
        annotations:
          eks.amazonaws.com/role-arn: arn:aws:iam::12345678910:role/eksctl-demo-cluster-addon-iamservic-Role1-J78KUNY32R1
  ```
  and then re-deploy 
  ```
  kubectl create -f ./my-falcon-container.yaml
  ```
