# CrowdStrike Falcon Operator Release Notes

## Release 0.7.2

* Sets default replica count of falcon injector to 2, and enables pod topology spread on the falcon-injector deployment
* Excludes kube-system when creating docker registry secrets

## Release 0.7.1

* Adds node.backend attribute, to configure Falcon Sensor in kernel or bpf mode
* Adds default trace logging value of none

## Release 0.7.0

Version 0.7.0 of the Falcon Operator introduces a significant rewrite of the Falcon Container Sensor Controller.  The Falcon Container Custom Resource Definition has changed quite significantly; users are advised to review the [Falcon Operator documentation for the Falcon Container Sensor](docs/container) before attempting to install this release, as some attributes have been changed or removed.

### Notable changes

* Falcon Container Sensor Controller no longer leverages the Falcon Container installer to generate Kubernetes manifests; resources are managed in-line within the Operator codebase
  * Resources managed by the Falcon Container Sensor Controller will now have any drift reconciled automatically
  * Logs no longer contain Kubernetes manifests of instantiated objects
  * Custom Resource Definition better documents user configurable options
  * installer_args has been deprecated and removed from the FalconContainer Custom Resource Definition
* Adjustments to the Falcon Operator Controller Runtime Manager Cache
  * Where prudent, utilizes selectors to minimize the resource impact of managing the lifecycle of multiple Kubernetes object types
