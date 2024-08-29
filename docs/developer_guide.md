# Developer Guide

## Tool Prerequisites

The following tools are required to develop the Falcon Operator:

- [git][git-tool]
- [go][go-tool] version 1.21
- [operator-sdk][operator-sdk] version 1.34.1
- [docker][docker] (required for multi-arch builds) or [podman][podman] (if desired for single arch builds)

Running `make` at any point will install additional tooling and go dependencies as required by the various `Makefile` targets. For example:

- kustomize
- controller-gen
- envtest

## Building

The various components of the operator can be built by running the `make` command. As there are various targets, run `make help` to display the supported `Makefile` targets.

### Direct Operator Deployment
To build and test changes for golang code changes, run the following commands:

```sh
make docker-build docker-push IMG="myregistry/crowdstrike/falcon-operator:test_tag"
```

Deploy the operator:

```sh
make deploy IMG="myregistry/crowdstrike/falcon-operator:test_tag"
```

Once done, remove the deployment:

```sh
make undeploy
```

### OLM Bundle Deployment
To build and test OLM Bundle changes for an OLM cluster, run the following commands on a running OLM-enabled cluster:

```sh
make bundle IMG="myregistry/crowdstrike/falcon-operator:v0.0.1"
make bundle-build bundle-push BUNDLE_IMG="myregistry/crowdstrike/falcon-operator-bundle:v0.0.1"
```

Then run the following `operator-sdk` commands to deploy the OLM bundle to a running OLM-enabled cluster:

```sh
operator-sdk run bundle myregistry/crowdstrike/falcon-operator-bundle:v0.0.1
```

Once you are done confirming changes, make sure to cleanup the deployment:
```sh
operator-sdk cleanup falcon-operator
```

## Testing

There are 2 type of tests that can be run: End-to-end (e2e) and Integration.

To run e2e testing, make sure that you are logged in to a running kubernetes cluster and run the following command:

```sh
make test-e2e
```

To run integration tests, run the following command:

```sh
make test
```

## Releasing

### Tagging a new release

1. Releasing is currently done on maintenance branches. Make sure to switch to the maintenance branch before tagging.
   `git checkout maint-1.2.3`
2. `git tag v1.2.3 && git push origin v1.2.3`
3. Wait several minutes for builds to run: <https://github.com/crowdstrike/falcon-operator/actions>. This run will not only create the release but also update resources and changelog in the repository itself.

If the build fails, there is no clean way to re-run the release action. The easiest way would be to start over by deleting the partial release on GitHub and re-publishing the tag.

## Continuous Integration (CI)

The Falcon Operator project uses GitHub Actions that run as part of pull request, merge, and release processes.
To test deployment against a KIND cluster, add the `ok-to-test` label to a pull request. Only project owners will be
able to do this step. Make sure to review the pull request first to ensure that the changes are valid and do not contain nefarious actions.

## Contribution flow

This is a rough outline of what a contributor's workflow looks like:

- Create a topic branch from where to base the contribution. This is usually the main branch.
- Make commits of logical units.
- Make sure commit messages are in the proper format (see below).
- Push changes in a topic branch to a personal fork of the repository.
- Submit a pull request.
- The PR must be reviewed by an owner and issues raised in open pull requests must be addressed.

Thanks for contributing!

### Code style

The coding style suggested by the Go community is used in operator-sdk. See the [style doc][golang-style-doc] for details.

### Format of the commit message

We follow a rough convention for commit messages that is designed to answer two
questions: what changed and why. The subject line should feature the what and
the body of the commit should describe the why. Sometime for small changes only setting
the subject line will suffice.

```
feature: add the test-cluster command

this uses a test cluster that can easily be killed and started for debugging.

Fixes #123
```

The format can be described more formally as follows:

```
<subsystem>: <what changed>
<BLANK LINE>
<why this change was made>
<BLANK LINE>
<footer>
```

The first line is the subject and should be no longer than 70 characters, the second line is always blank, and other lines should be wrapped at 80 characters. This allows the message to be easier to read on GitHub as well as in various git tools.

[git-tool]:https://git-scm.com/downloads
[go-tool]:https://golang.org/dl/
[operator-sdk]:https://sdk.operatorframework.io/docs/installation/
[golang-style-doc]: https://github.com/golang/go/wiki/CodeReviewComments
[docker]:https://docs.docker.com/engine/install/
[podman]:https://podman.io/getting-started/installation
