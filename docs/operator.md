# Troubleshooting

To review the logs of Falcon Operator:
```
kubectl -n falcon-operator logs -f deploy/falcon-operator-controller-manager -c manager
```

## Operator Issues

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
