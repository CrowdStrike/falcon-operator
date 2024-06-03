# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.0.0] - 2024-06-03

### Changed

- build(deps): bump golangci/golangci-lint-action from 5 to 6
- docs: update docs for iar and openshift
- chore(bundle): add arm64 support label
- cleanup(bundle): remove legacy unused falconcontainer role
- --- updated-dependencies: - dependency-name: github.com/docker/docker   dependency-type: indirect ...
- --- updated-dependencies: - dependency-name: github.com/containers/image/v5   dependency-type: direct:production ...
- Bumping to version 0.9.6
- regenerate boilerplate code
- add docs
- more code cleanup
- configure ocp scc for iar
- cleanup old iar code
- exclude docs in gosec testing
- configure volumesize before default is generated
- make volume and volumemount configuration simpler
- update IAR types for exclusions and registry configs, etc.
- Remove Falcon sensor settings for IAR
- add volumes and volumemount configs
- remove unused functions and add azureconfig and priorityclassname to config
- security context
- minor fix
- non-olm
- cleanup configmap
- add bundle
- add manager config
- remove unused
- generate manifest and api
- add imageanalyzer role
- remove resourceQuota
- cleanup deployment
- cleanup controller
- image tag
- update rbac
- add configmap
- lint
- update manifests
- IAR controller and templates
- falcon image deployment function
- falcon image constants
- falcon image type definitions
- cleanup: conditionsupdate should get resource
- cleanup: remove tautological conditions
- cleanup: remove unused parameters
- feat: allow sidecar sensor to customize namespace
- chore: add arch requirements for single-arch deployments
- feat(admission): automate ocp & falcon ns exclusions
- docs(nodesensor): update docs
- fix(nodesensor): use operator naming for node clusterrolebinding
- chore(nodesensor): add privileged labels to node sensor ns
- feat(sec): do not allow any workloads to run in falcon install namespaces
- feat: allow node sensor to customize namespace
- chore: use retry on conflict to update the status
- build(deps): bump golangci/golangci-lint-action from 4 to 5
- build(deps): bump helm/kind-action from 1.9.0 to 1.10.0
- fix(admission): version 7.14 of admission controller requires webhook to exist before the service can start
- feat(node): handle multi-arch container images
- feat: operator upgrade docs
- chore(admission): sync clusterrole perms
- fix src
- build(deps): bump golang.org/x/net from 0.21.0 to 0.23.0
- fix admission readme
- make tags array
- chore(action): update metadata action to add non-olm target
- fix(ci): fix broken tests due to upstream envtest changes
- build(deps): bump github.com/docker/docker
- ran make and added generated files
- imagePullSecretName is no longer valid, updated readme with imagePullSecret
- fix(iar): return IAR tags
- cleanup(nodesensor): remove legacy initContainer code
- build(deps): bump google.golang.org/protobuf from 1.31.0 to 1.33.0
- feat: determine cluster install features such as OpenShift and cert-manager
- build(deps): bump github.com/go-jose/go-jose/v3 from 3.0.1 to 3.0.3
- build(deps): bump gopkg.in/go-jose/go-jose.v2 from 2.6.1 to 2.6.3
- cleanup: Remove WATCH_NAMESPACE usage
- feat: update to operator-sdk 1.34.1
- fix(admission): always return existing tls certs on reconciliation
- Clarify FalconContainer is not intended for OpenShift.
- fix(admission): Fix admission controller yaml for azure
- feat: update gofalcon to v0.6.0
- feat: remove CGO_ENABLED=0 references in order to enable FIPS compliance
- build(deps): bump helm/kind-action from 1.8.0 to 1.9.0
- build(deps): bump golangci/golangci-lint-action from 3 to 4
- feat(node): merge tolerations when injected
- docs: add managed OpenShift control plan/infra caveats
- feat: add new OpenShift feature annotations to CSV
- fix: remove deprecated configmap for componentconfig
- fix: update leader election ID
- fix: update oom guidance for openshift to makes changes to the subscription
- fix: operator pull policy should follow the default
- feat: add some utils tests
- fix: TestMakeSensorEnvMap should test for automatic proxy vs manual
- fixing the automatic proxy host config commenting the test TestMakeSensorEnvMapWithAutomaticProxy for refactor
- fix: priorityclass handling should be deployable to more than just GKE
- feat: OLM updates
- feat: generate boilerplate for IAR
- fix: downloaded kustomize if needed when non-olm make target is run
- build(deps): bump github.com/opencontainers/runc from 1.1.10 to 1.1.12
- cleanup: remove logging from version.go
- Bump channel in docs/src.
- Fix a readme link so it works from OperatorHub.
- Bump OpenShift Subscription channel to 0.9.
- cleanup: remove cloudformation content
- fix: update go crypto version
- build(deps): bump github.com/containerd/containerd from 1.7.0 to 1.7.11
- fix: update manifests
- fix: update controller-runtime cache handling from deprecated method
- fix: go mod tidy
- feat: use gofalcon for registry config and sensor types
- fix: use valid yaml sequence
- fix: remove deprecated componentConfig and controller manager options
- feat: migrate controllers to new folder to match golang project standards
- fix: use LOCALBIN for opm install
- build(deps): bump github/codeql-action from 2 to 3
- build(deps): bump actions/setup-go from 4 to 5
- update CRD to fix the display name on the proxy host
- fix: checkout branch to get release commit during release run
- feat: update to operator-sdk version 1.33.0
- feat: set operator to be permanently globally scoped
- feat: add infra node toleration by default
- clean(node-sensor): remove some unnecessary functions
- feat: Add network permissions for GKE Autopilot
- feat: update to latest gofalcon
- feat: loosen up the default resource quota the admission controller
- fix: admission controller doc fixes
- fix: fix typo in configmap_test.go

## [0.9.6] - 2024-05-10

### Changed

- fix(ci): fix broken tests due to upstream envtest changes
- feat: support multi-arch cs images

## [0.9.5] - 2024-03-14

### Changed

- build(deps): bump google.golang.org/protobuf from 1.31.0 to 1.33.0
- build(deps): bump gopkg.in/go-jose/go-jose.v2 from 2.6.1 to 2.6.3
- cleanup(nodesensor): remove legacy initContainer code

## [0.9.4] - 2024-03-07

### Changed

- Clarify FalconContainer is not intended for OpenShift.
- fix(admission): Fix admission controller yaml for azure
- docs: add managed OpenShift control plan/infra caveats
- feat: add new OpenShift feature annotations to CSV
- fix: update oom guidance for openshift to makes changes to the subscription
- fix(admission): always return existing tls certs on reconciliation

## [0.9.3] - 2024-02-08

### Changed

- fix: TestMakeSensorEnvMap should test for automatic proxy vs manual
- fixing the automatic proxy host config commenting the test TestMakeSensorEnvMapWithAutomaticProxy for refactor
- fix: priorityclass handling should be deployable to more than just GKE
- fix: downloaded kustomize if needed when non-olm make target is run
- Bump channel in docs/src.
- Fix a readme link so it works from OperatorHub.
- Bump OpenShift Subscription channel to 0.9.
- fix: update go crypto version
- update CRD to fix the display name on the proxy host
- feat: update to latest gofalcon
- feat: loosen up the default resource quota the admission controller
- fix: admission controller doc fixes
- fix: fix typo in configmap_test.go

## [0.9.2] - 2023-12-22

### Changed

- feat: add infra node toleration by default
- fix: checkout branch to get release commit during release run
- feat: Add network permissions for GKE Autopilot

## [0.9.1] - 2023-11-03

### Changed

- fix: sensor resource handling

## [0.9.0] - 2023-11-01

### Changed

- feat: update proxy section and add sensor upgrade section
- fix: add node lock
- feat: update falconadmission resource
- feat: update readme with falconadmission resource
- feat: add resources to initContainer and cleanup
- feat: add Admission Controller docs
- build(deps): bump github.com/docker/docker
- fix: use GH alert formatting
- feat: add gke autopilot docs
- feat: enable GKE autopilot support
- build(deps): bump google.golang.org/grpc from 1.55.0 to 1.56.3
- fix: update operator and image version status when changed
- feat: update bundle for admission controller
- fix: various test issues
- feat: Add admission controller test suite
- fix: ensure operator management config for non-OpenShift distros
- feat: enable FIPS-capable container builds
- feat: add admission controller reconciler
- feat: Update kustomize scaffolding for admission controller settings
- feat: add admission controller deployment
- feat: update proxy docs to provide link and examples
- fix: node sensor tolerations are stuck in constant update
- feat: add sidecar e2e test run
- build(deps): bump golang.org/x/net from 0.10.0 to 0.17.0
- feat: add Sidecar controller test
- feat: support admission controller registry
- fix: various scaffolding fixes
- feat: add common reconciliation functions to cut down on code duplication
- feat: update service asset to pass service name
- feat: add admission controller RBAC config
- fix: config sample fixes
- feat: get args from env for OLM config
- build(deps): bump docker/setup-buildx-action from 2 to 3
- build(deps): bump docker/setup-qemu-action from 2 to 3
- build(deps): bump docker/build-push-action from 4 to 5
- build(deps): bump actions/checkout from 3 to 4
- build(deps): bump docker/login-action from 2 to 3
- feat: run doc tests from makefile
- feat: add linting to Makefile
- feat: Generate docs from templates
- feat: add GH Action to error when autogenerated docs are changed manually
- feat: Add initial scaffolding for helm chart source
- build(deps): bump github.com/cyphar/filepath-securejoin
- refactor: code re-use for certs, pods ready check, ImageRefresher, etc.
- build(deps): bump helm/kind-action from 1.7.0 to 1.8.0
- feat: enable MaxSurge in DS
- fix: consistently use falconv1alpha1 for falcon v1alpha1 imports
- fix: FalconAdmission boilerplate fixes
- feat: add admission controller scaffolding
- feat: start to use internal/controller and dedup some Kinds
- fix: update api dir for golang standards structure
- fix: update main.go to follow golang dir standards structure
- feat: update config to SDK version 1.30
- makefile: update to the latest operator-sdk and kubebuilder versions
- feat: update to golang 1.19
- feat: add proxy support
- bump version to 0.9.0
- fix: update changelog with 0.8.1 changes


## [0.8.1] - 2023-06-07

### Changed

- Bump version to 0.8.1
- build(deps): bump github.com/sigstore/rekor from 1.1.0 to 1.2.0
- build(deps): bump github.com/docker/docker
- maint: go mod tidy
- maint: update changelog
- feat: standardize labels across controllers
- fix: update docs for new release
- fix: delay CS registry API check for falconcontainer
- build(deps): bump helm/kind-action from 1.5.0 to 1.7.0
- fix: sidecar deployment should have a service account specified
- docs: update redhat deployment doc and images
- fix: update CSV description
- docs: doc updates
- fix: various fixes in prep for future changes
- cleanup: create a common label function
- fix: various fixes and certification prep
- fix: sensor version was not working correctly
- fix: ensure custom non-API Falcon CID can be used
- fix: update runc go.mod indirect dependency
- Update README.md
- cmm edits to clean up verbiage and look/feel
- feat: Add Krew instructions and update OCP instructions
- feat: create generic kubernetes install
- fix: update indirect runc dependency to version 1.1.5
- fix: Makefile kustomize target
- docs: resource docs updates
- Add operatorgroup and some troubleshooting steps
- GKE, EKS, Azure updates
- OCP image updates and node doc updates
- Documentation updates
- fix: fix Makefile help output for 2 targets
- feat: make developer guide more robust
- fix: update metadata to use release version
- fix: disable seccompProfile until broadly supported and enable multi-arch affinity for controller-manager
- fix: update tags for release automation
- fix: reconciliation loop should not run forever
- fix: fix failing deployment tests
- feat: automate releases
- fix: use released manifests for non-olm deployments
- fix: update CSV contact info
- build(deps): bump github.com/docker/docker
- build(deps): bump actions/setup-go from 3 to 4
- feat: add support for nodeAffinity in node sensor
- fix: cluster role and SCC should not be reconciled
- fix: update DS labels
- Adding release note
- fix: update deployment on replica count change
- feat: add docker release build
- fix: provide more test coverage in node assets
- clean up ds updates
- fix test cleanup args
- update tests
- clean up updates
- fix: re-organize go workflows
- fix: Update falcon-operator.yaml
- fix: Update labels in assets
- feat: add labels, security, and arch affinity to kustomize components
- fix: Dockerfile cross compile updates and Makefile updates
- fix: ensure non-olm deployment uses kustomize serviceaccount
- Update falcon-operator.yaml using kustomize
- feat: Use kustomize to generate non-olm package manifest
- fix: kustomize format operator non-olm deploy yaml
- node: updating init containers for node daemonset and node cleanup daemonset
- Update README.md
- Update README.md
- Log the falcon node sensor image uri selected to be used
- feat: Enable multi-arch operator build
- build(deps): bump golang.org/x/net from 0.1.0 to 0.7.0
- update bundle
- update pod topology and replica count
- fix: exclude gosec rule G307 as it has been removed in the upstream branch
- build(deps): bump helm/kind-action from 1.4.0 to 1.5.0
- Explicitly excluding kube-system from secret creation
- Adding documentation for node.backend
- bump CSV version
- Bumping version to 0.7.1
- Adding backend support in Node/DaemonSet
- Do not deploy status: subresources outside OLM
- remove falconctlOpts to use default properties
- update bundle manifests
- update properties in the readme for Node and Container
- Update FalconContainer All options with default falcon values
- adding default trace value in the yaml
- fixing values and typo

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

Version 0.7.0 of the Falcon Operator introduces a significant rewrite of the Falcon Container Sensor Controller.  The Falcon Container Custom Resource Definition has changed quite significantly; users are advised to review the [Falcon Operator documentation for the Falcon Container Sensor](docs/resources/container) before attempting to install this release, as some attributes have been changed or removed.

### Notable changes

* Falcon Container Sensor Controller no longer leverages the Falcon Container installer to generate Kubernetes manifests; resources are managed in-line within the Operator codebase
  * Resources managed by the Falcon Container Sensor Controller will now have any drift reconciled automatically
  * Logs no longer contain Kubernetes manifests of instantiated objects
  * Custom Resource Definition better documents user configurable options
  * installer_args has been deprecated and removed from the FalconContainer Custom Resource Definition
* Adjustments to the Falcon Operator Controller Runtime Manager Cache
  * Where prudent, utilizes selectors to minimize the resource impact of managing the lifecycle of multiple Kubernetes object types
