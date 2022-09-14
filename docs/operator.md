# Troubleshooting

To review the logs of Falcon Operator:
```
kubectl -n falcon-operator logs -f deploy/falcon-operator-controller-manager -c manager
```

## Operator Issues

#### Configure to watch only the `falcon-system` namespace when using FalconNodeSensor

Since the Falcon Operator is a global-scoped operator, it watches across resources and objects across all namespaces.
For large clusters that might have large configmaps and secrets, this requires a lot of memory to be consumed which may be less than ideal.
If you do not want the operator to watch globally when using the FalconNodeSensor resource, configure the operator to only watch the `falcon-system` namespace
Please note that the following settings should not be configured when using the FalconContainer Resource.

##### OpenShift CLI

- If this is a brand new install and you are using the `oc` cli tool, create the following file:
  ```
  cat << EOF >> operatorgroup.yaml

  apiVersion: operators.coreos.com/v1
  kind: OperatorGroup
  metadata:
    name: falcon-operator-og
  spec:
    targetNamespaces:
    - falcon-system
  EOF
  ```
  The `OperatorGroup` will tell the operator to only watch a single namespace. It will also persist independently of the operator so that the setting will apply whenever 
  the operator is installed, un-installed, or upgraded.

- Create the Falcon Operator subscription to install from the console:
  ```
  cat << EOF >> sub.yaml
  apiVersion: operators.coreos.com/v1alpha1
  kind: Subscription
  metadata:
    name: falcon-operator-v0-5-4-sub
  spec:
    channel: alpha
    name: falcon-operator
    source: community-operators
    sourceNamespace: openshift-marketplace
    startingCSV: falcon-operator.v0.5.4
  EOF
  ```

- Deploy the operator
  ```
  $ oc create -f operatorgroup.yaml -n falcon-operator
  $ oc create -f sub.yaml -n falcon-operator
  ```
  The operator should now deploy while only watching the `falcon-system` namespace.

- If you have already installed the operator, edit the auto-generated `OperatorGroup`:
  ```
  $ oc edit operatorgroup -n falcon-operator
  ```

- Add the following by replacing `spec: {}` with:
  ```
  spec: 
    targetNamespaces:
      - falcon-system
  ```
  Save the changes and the operator deployment will update and rollout a new deployment.

##### OpenShift GUI

##### EKS, AKS, GKE, and non-OpenShift or non-OLM installs

- Configure the `WATCH_NAMESPACE` env variable by editing the Falcon Operator deployment configuration:
  ```
  $ kubectl edit deployment falcon-operator-controller-manager -n falcon-operator
  ```

- Add the `value` of `falcon-system` to the `WATCH_NAMESPACE` env variable, for example:
  ```
  - name: WATCH_NAMESPACE
    value: 'falcon-system'
  ```
  Save the changes and the operator deployment will update and rollout a new deployment.

#### ERROR setup failed to get watch namespace

If the following error shows up in the controller manager logs:
```
1.650281912313243e+09 ERROR setup failed to get watch namespace {"error": "WATCH_NAMESPACE must be set"}
1.6502819123132205e+09 INFO version go {"version": "go1.17.9 linux/amd64"}
1.6502819123131733e+09 INFO version operator {"version": "0.5.0-de97605"}
```
Make sure that the environment variable exists in the controller manager deployment. If it does not exist, add it by running:
```
kubectl edit deployment falcon-operator-controller-manager -n falcon-operator
```
and add something similar to the following lines:
```
        env:
          - name: WATCH_NAMESPACE
            value: ''
          - name: POD_NAME
            valueFrom:
              fieldRef:
                fieldPath: metadata.name
          - name: OPERATOR_NAME
            value: "falcon-operator"
```

#### FalconContainer is stuck in the CONFIGURING Phase

Make sure that the `WATCH_NAMESPACE` variable is correctly configured to be cluster-scoped and not namespace-scoped. If the 
controller manager's deployment has the following configuration:
```
        env:
          - name: WATCH_NAMESPACE
            valueFrom:
              fieldRef:
                fieldPath: metadata.namespace
```
the operator is configured to be namespace-scoped and not cluster-scoped which is required for the FalconContainer CR.
This problem can be fixed by running:
```
kubectl edit deployment falcon-operator-controller-manager -n falcon-operator
```
and changing `WATCH_NAMESPACE` to the following lines:
```
        env:
          - name: WATCH_NAMESPACE
            value: ''
          - name: POD_NAME
            valueFrom:
              fieldRef:
                fieldPath: metadata.name
          - name: OPERATOR_NAME
            value: "falcon-operator"
```
Once a new version of the controller manager has deployed, you may have to delete and recreate the FalconContainer CR.
