# Developer Guide

## Set-up environment variables for your workspace

```
    REGISTRY_HOST=quay.io
    REGISTRY_LOGIN=johndoe
    export OPERATOR_IMG=$REGISTRY_HOST/$REGISTRY_LOGIN/devel-falcon-operator:v0.0.1
    export BUNDLE_IMG=$REGISTRY_HOST/$REGISTRY_LOGIN/devel-falcon-operator-bundle:v0.0.1
```

## Build and push the images

```
    make docker-build bundle bundle-build docker-push IMG=$OPERATOR_IMG BUNDLE_IMG=$BUNDLE_IMG
    docker push $BUNDLE_IMG
```

## Install the operator


```
    NAMESPACE=falcon-operator
    operator-sdk run bundle $BUNDLE_IMG --namespace $NAMESPACE
    kubectl wait --timeout=240s --for=condition=Available -n $NAMESPACE deployment falcon-operator-controller-manager
```
