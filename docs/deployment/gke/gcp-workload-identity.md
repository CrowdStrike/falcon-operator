# Granting GCP Workload Identity to Falcon Container Injector

Falcon Container Injector may need [GCP Workload Identity](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity)
to read GCR or Artifact Registry. In many cases this GCP Workload Identity is assigned or inherited automatically. However, if you
are seeing errors similar to the following you may need to follow this guide to assign the identity manually.

```
time="2022-01-14T13:05:11Z" level=error msg="Failed to handle webhook request" error="Failed to retrieve image details for \"gcr.io/\" in container in pod: Failed to get the image config/digest for \"gcr.io/" on \"eu.gcr.io\": Error reading manifest latest in gcr.io/: unauthorized: You don't have the needed permissions to perform this operation, and you may have invalid credentials. To authenticate your request, follow the steps in: https://cloud.google.com/container-registry/docs/advanced-authentication"
```

## Assigning GCP Workload Identity to Falcon Container Injector

Conceptually, the following tasks need to be done in order to enable GCR pull from the injector

 - Create GCP Service Account
 - Grant GCR permissions to the newly created Service Account
 - Allow Falcon Container to use the newly created Service Account
 - Put GCP Service Account handle into your Falcon Container resource for re-deployments

The following step-by-step guide uses `gcloud`, and `kubectl` command-line tools to achieve that.

## Step-by-step guide

 - Set up your shell environment variables
   ```
   GCP_SERVICE_ACCOUNT=falcon-container-injector

   GCP_PROJECT_ID=$(gcloud config get-value core/project)
   ```

 - Create new GCP Service Account
   ```
   gcloud iam service-accounts create $GCP_SERVICE_ACCOUNT
   ```

 - Grant GCR permissions to the newly created Service Account
   ```
   gcloud projects add-iam-policy-binding $GCP_PROJECT_ID \
       --member "serviceAccount:$GCP_SERVICE_ACCOUNT@$GCP_PROJECT_ID.iam.gserviceaccount.com" \
       --role roles/containerregistry.ServiceAgent 
   ```
   
 - Allow Falcon Injector to use the newly created GCP Service Account
   ```
   gcloud iam service-accounts add-iam-policy-binding \
       $GCP_SERVICE_ACCOUNT@$GCP_PROJECT_ID.iam.gserviceaccount.com \
       --role roles/iam.workloadIdentityUser \
       --member "serviceAccount:$GCP_PROJECT_ID.svc.id.goog[falcon-system/default]"
   ```

- Re-deploy (delete & create) FalconContainer with the above Service Account added to the spec:

  Delete FalconContainer
  ```
  kubectl delete falconcontainers --all
  ```

  Add Newly created Service Account to your FalconContainer yaml file:
  ```
  spec:
    injector:
      serviceAccount:
        annotations:
          iam.gke.io/gcp-service-account: $GCP_SERVICE_ACCOUNT@$GCP_PROJECT_ID.iam.gserviceaccount.com
  ```
  
  (don't forget to replace the service account name template with actual name)
  ```
  echo "$GCP_SERVICE_ACCOUNT@$GCP_PROJECT_ID.iam.gserviceaccount.com"
  ```
