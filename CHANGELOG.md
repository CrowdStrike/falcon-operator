# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.7.3] - 2023-04-14

### Changed

- fix: update tags for release automation
- clean up ds updates
- fix test cleanup args
- update tests
- clean up updates
- node: updating init containers for node daemonset and node cleanup daemonset
- build(deps): bump github.com/docker/docker
- fix: fix version in Makefile for non-OLM manifest
- feat: automate releases
- feat: add docker release build
- fix: use released manifests for non-olm deployments
- Fix build.sh and Makefile to build for target architecture
- Fix bundle image reference
- Fix bundle image reference
- Limit daemonset image lookup to current architecture
- Update Makefile buildx targets
- Bump operator version and fix image reference
- Releasing 0.7.1
- Adding release note
- fix: update deployment on replica count change
- fix: provide more test coverage in node assets
- fix: re-organize go workflows
- fix: Dockerfile cross compile updates and Makefile updates
- Update README.md
- Update README.md
- Log the falcon node sensor image uri selected to be used
- feat: Enable multi-arch operator build
- build(deps): bump golang.org/x/net from 0.1.0 to 0.7.0
- update bundle
- update pod topology and replica count
- fix: exclude gosec rule G307 as it has been removed in the upstream branch
- build(deps): bump helm/kind-action from 1.4.0 to 1.5.0
- Do not deploy status: subresources outside OLM
- Explicitly excluding kube-system from secret creation
- Adding documentation for node.backend
- bump CSV version
- Bumping version to 0.7.1
- Adding backend support in Node/DaemonSet
- remove falconctlOpts to use default properties
- update bundle manifests
- update properties in the readme for Node and Container
- Update FalconContainer All options with default falcon values
- adding default trace value in the yaml
- fixing values and typo

## [0.7.2] - 2023-03-29

### Changed

* Sets default replica count of falcon injector to 2, and enables pod topology spread on the falcon-injector deployment
* Excludes kube-system when creating docker registry secrets

## [0.7.1] - 2022-12-08

### Changed

* Adds node.backend attribute, to configure Falcon Sensor in kernel or bpf mode
* Adds default trace logging value of none

## [0.7.0] - 2022-12-01

### Changed

Version 0.7.0 of the Falcon Operator introduces a significant rewrite of the Falcon Container Sensor Controller.  The Falcon Container Custom Resource Definition has changed quite significantly; users are advised to review the [Falcon Operator documentation for the Falcon Container Sensor](docs/container) before attempting to install this release, as some attributes have been changed or removed.

### Notable changes

* Falcon Container Sensor Controller no longer leverages the Falcon Container installer to generate Kubernetes manifests; resources are managed in-line within the Operator codebase
  * Resources managed by the Falcon Container Sensor Controller will now have any drift reconciled automatically
  * Logs no longer contain Kubernetes manifests of instantiated objects
  * Custom Resource Definition better documents user configurable options
  * installer_args has been deprecated and removed from the FalconContainer Custom Resource Definition
* Adjustments to the Falcon Operator Controller Runtime Manager Cache
  * Where prudent, utilizes selectors to minimize the resource impact of managing the lifecycle of multiple Kubernetes object types
