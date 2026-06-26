# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

# ============================================================================
##@ Project Configuration
# ============================================================================

# VERSION defines the project version for the bundle.
# Update this value when you upgrade the version of your project.
VERSION ?= 1.5.0

# Image URL to use all building/pushing image targets
IMG ?= controller:latest

# IMAGE_TAG_BASE defines the docker.io namespace and part of the image name for remote images.
IMAGE_TAG_BASE ?= crowdstrike/falcon-operator

# ============================================================================
##@ Build Configuration
# ============================================================================

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# GOFLAGS defines additional golang build flags like -tags and -ldflags
GOFLAGS ?= -a \
         -tags "exclude_graphdriver_devicemapper exclude_graphdriver_btrfs containers_image_openpgp" \
         --ldflags="-X 'github.com/crowdstrike/falcon-operator/version.Version=$(VERSION)'"

# CONTAINER_TOOL defines the container tool to be used for building images.
CONTAINER_TOOL ?= docker

# GOPROXY defines the Go module proxy to use for downloading dependencies
GOPROXY ?= https://proxy.golang.org,direct

# CONTAINER_BUILD_ARGS defines additional build arguments to pass to the $CONTAINER_TOOL during build.
CONTAINER_BUILD_ARGS ?= --build-arg VERSION=$(VERSION) --build-arg GOPROXY=$(GOPROXY)

# PLATFORMS defines the target platforms for the manager image be built to provide support to multiple architectures.
PLATFORMS ?= linux/arm64,linux/amd64,linux/s390x,linux/ppc64le

# ============================================================================
##@ Bundle/OLM Configuration
# ============================================================================

# CHANNELS define the bundle channels used in the bundle.
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif

# DEFAULT_CHANNEL defines the default channel used in the bundle.
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# BUNDLE_IMG defines the image:tag used for the bundle.
BUNDLE_IMG ?= $(IMAGE_TAG_BASE)-bundle:v$(VERSION)

# BUNDLE_GEN_FLAGS are the flags passed to the operator-sdk generate bundle command
BUNDLE_GEN_FLAGS ?= -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)

# USE_IMAGE_DIGESTS defines if images are resolved via tags or digests
USE_IMAGE_DIGESTS ?= false
ifeq ($(USE_IMAGE_DIGESTS), true)
	BUNDLE_GEN_FLAGS += --use-image-digests
endif

# The image tag given to the resulting catalog image
CATALOG_IMG ?= $(IMAGE_TAG_BASE)-catalog:v$(VERSION)

# A comma-separated list of bundle images
BUNDLE_IMGS ?= $(BUNDLE_IMG)

# Set CATALOG_BASE_IMG to an existing catalog image tag to add $BUNDLE_IMGS to that image.
ifneq ($(origin CATALOG_BASE_IMG), undefined)
FROM_INDEX_OPT := --from-index $(CATALOG_BASE_IMG)
endif

# ============================================================================
##@ Tool Configuration
# ============================================================================

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p "$(LOCALBIN)"

## Tool Binaries
KUBECTL ?= kubectl
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
KUBEBUILDER ?= $(LOCALBIN)/kubebuilder
OPERATOR_SDK ?= $(LOCALBIN)/operator-sdk
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint
OPM ?= $(LOCALBIN)/opm

## Tool Versions
KUSTOMIZE_VERSION ?= v5.8.1
CONTROLLER_TOOLS_VERSION ?= v0.20.1
KUBEBUILDER_VERSION ?= v4.15.0
OPERATOR_SDK_VERSION ?= v1.38.0
GOLANGCI_LINT_VERSION ?= v1.62.2

## Auto-detected versions from go.mod
# ENVTEST_VERSION is the version of controller-runtime release branch (i.e. release-0.20)
ENVTEST_VERSION ?= $(shell v="$$(hack/gomodver.sh sigs.k8s.io/controller-runtime)"; \
  [ -n "$$v" ] || { echo "Set ENVTEST_VERSION manually (controller-runtime replace has no tag)" >&2; exit 1; }; \
  printf '%s\n' "$$v" | sed -E 's/^v?([0-9]+)\.([0-9]+).*/release-\1.\2/')

# ENVTEST_K8S_VERSION is the version of Kubernetes to use for setting up ENVTEST binaries (i.e. 1.31)
ENVTEST_K8S_VERSION ?= $(shell v="$$(hack/gomodver.sh k8s.io/api)"; \
  [ -n "$$v" ] || { echo "Set ENVTEST_K8S_VERSION manually (k8s.io/api replace has no tag)" >&2; exit 1; }; \
  printf '%s\n' "$$v" | sed -E 's/^v?[0-9]+\.([0-9]+).*/1.\1/')

# ============================================================================
##@ General
# ============================================================================

.PHONY: all
all: build

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

# ============================================================================
##@ Development
# ============================================================================

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	"$(CONTROLLER_GEN)" rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	"$(CONTROLLER_GEN)" object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: manifests generate fmt vet setup-envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell "$(ENVTEST)" use $(ENVTEST_K8S_VERSION) --bin-dir "$(LOCALBIN)" -p path)" go test $$(go list ./... | grep -v /test/) -coverprofile cover.out

# ============================================================================
##@ Testing
# ============================================================================

# Environment Variables for e2e tests:
#   USE_EXISTING_OPERATOR=true - Skip operator installation and use existing deployment
#   OPERATOR_NAMESPACE=falcon-system - Namespace where operator is deployed (with USE_EXISTING_OPERATOR)
#   BUNDLE_IMG=<image> - Use OLM bundle installation instead of traditional deployment
#   OPERATOR_IMAGE=<image> - Custom operator image to use
#   GINKGO_LABEL_FILTER=<filter> - Filter tests by label (e.g., "!FalconNodeSensor")
#   GINKGO_FOCUS=<regex> - Focus on tests matching this regular expression
#   GINKGO_SKIP=<regex> - Skip tests matching this regular expression
#   FALCON_CLIENT_ID - CrowdStrike Falcon API Client ID
#   FALCON_CLIENT_SECRET - CrowdStrike Falcon API Client Secret
USE_EXISTING_OPERATOR ?= false

.PHONY: test-e2e
test-e2e: operator-sdk ## Run e2e tests against a Kind k8s instance or existing operator installation
	@echo "Downloading test dependencies with GOPROXY=$(GOPROXY)..."
	@GOPROXY=$(GOPROXY) go mod download
	@set -e; \
	export GOPROXY=$(GOPROXY); \
	GINKGO_ARGS="-v -ginkgo.v -timeout 30m"; \
	if [ -n "$(GINKGO_LABEL_FILTER)" ]; then \
		GINKGO_ARGS="$$GINKGO_ARGS -ginkgo.label-filter='$(GINKGO_LABEL_FILTER)'"; \
	fi; \
	if [ -n "$(GINKGO_FOCUS)" ]; then \
		GINKGO_ARGS="$$GINKGO_ARGS -ginkgo.focus='$(GINKGO_FOCUS)'"; \
	fi; \
	if [ -n "$(GINKGO_SKIP)" ]; then \
		GINKGO_ARGS="$$GINKGO_ARGS -ginkgo.skip='$(GINKGO_SKIP)'"; \
	fi; \
	if [ "$(USE_EXISTING_OPERATOR)" = "true" ]; then \
		USE_EXISTING_OPERATOR=true OPERATOR_NAMESPACE=$(or $(OPERATOR_NAMESPACE),falcon-system) \
		eval "go test ./test/e2e/ $$GINKGO_ARGS"; \
	else \
		eval "go test ./test/e2e/ $$GINKGO_ARGS"; \
	fi

.PHONY: lint
lint: golangci-lint ## Run golangci-lint linter
	"$(GOLANGCI_LINT)" run

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint linter and perform fixes
	"$(GOLANGCI_LINT)" run --fix

# ============================================================================
##@ Build
# ============================================================================

.PHONY: build
build: manifests generate fmt vet ## Build manager binary.
	go build $(GOFLAGS) -o bin/manager cmd/main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	go run ./cmd/main.go

.PHONY: docker-build
docker-build: ## Build docker image with the manager.
	$(CONTAINER_TOOL) build $(CONTAINER_BUILD_ARGS) -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	$(CONTAINER_TOOL) push ${IMG}

.PHONY: docker-buildx
docker-buildx: ## Build and push docker image for the manager for cross-platform support
	# copy existing Dockerfile and insert --platform=${BUILDPLATFORM} into Dockerfile.cross, and preserve the original Dockerfile
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' Dockerfile > Dockerfile.cross
	- $(CONTAINER_TOOL) buildx create --name project-v3-builder
	$(CONTAINER_TOOL) buildx use project-v3-builder
	- $(CONTAINER_TOOL) buildx build --push --platform=$(PLATFORMS) $(CONTAINER_BUILD_ARGS) --provenance=false --tag ${IMG} -f Dockerfile.cross .
	- $(CONTAINER_TOOL) buildx rm project-v3-builder
	rm Dockerfile.cross

# ============================================================================
##@ Deployment
# ============================================================================

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	@out="$$( "$(KUSTOMIZE)" build config/crd 2>/dev/null || true )"; \
	if [ -n "$$out" ]; then echo "$$out" | "$(KUBECTL)" apply -f -; else echo "No CRDs to install; skipping."; fi

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	@out="$$( "$(KUSTOMIZE)" build config/crd 2>/dev/null || true )"; \
	if [ -n "$$out" ]; then echo "$$out" | "$(KUBECTL)" delete --ignore-not-found=$(ignore-not-found) -f -; else echo "No CRDs to delete; skipping."; fi

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && "$(KUSTOMIZE)" edit set image controller=${IMG}
	"$(KUSTOMIZE)" build config/default | "$(KUBECTL)" apply -f -

.PHONY: undeploy
undeploy: kustomize ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	"$(KUSTOMIZE)" build config/default | "$(KUBECTL)" delete --ignore-not-found=$(ignore-not-found) -f -

# ============================================================================
##@ Tool Installation
# ============================================================================

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	@hack/install-kustomize.sh $(KUSTOMIZE) $(KUSTOMIZE_VERSION)

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	@hack/install-controller-gen.sh $(CONTROLLER_GEN) $(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download setup-envtest locally if necessary.
$(ENVTEST): $(LOCALBIN)
	@hack/install-envtest.sh $(ENVTEST) $(ENVTEST_VERSION)

.PHONY: setup-envtest
setup-envtest: envtest ## Download the binaries required for ENVTEST in the local bin directory.
	@echo "Setting up envtest binaries for Kubernetes version $(ENVTEST_K8S_VERSION)..."
	@"$(ENVTEST)" use $(ENVTEST_K8S_VERSION) --bin-dir "$(LOCALBIN)" -p path || { \
		echo "Error: Failed to set up envtest binaries for version $(ENVTEST_K8S_VERSION)."; \
		exit 1; \
	}

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	@hack/install-golangci-lint.sh $(GOLANGCI_LINT) $(GOLANGCI_LINT_VERSION)

.PHONY: operator-sdk
operator-sdk: $(OPERATOR_SDK) ## Download operator-sdk locally if necessary.
$(OPERATOR_SDK): $(LOCALBIN)
	@hack/install-operator-sdk.sh $(OPERATOR_SDK) $(OPERATOR_SDK_VERSION)

.PHONY: kubebuilder
kubebuilder: $(KUBEBUILDER) ## Download kubebuilder locally if necessary.
$(KUBEBUILDER): $(LOCALBIN)
	@hack/install-kubebuilder.sh $(KUBEBUILDER) $(KUBEBUILDER_VERSION)

.PHONY: opm
opm: $(OPM) ## Download opm locally if necessary.
$(OPM): $(LOCALBIN)
ifeq (,$(wildcard $(OPM)))
ifeq (,$(shell which opm 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPM)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sSLo $(OPM) https://github.com/operator-framework/operator-registry/releases/download/v1.23.0/$${OS}-$${ARCH}-opm ;\
	chmod +x $(OPM) ;\
	}
else
	@ln -sf $(shell which opm) $(OPM)
endif
endif

# ============================================================================
##@ Bundle/OLM
# ============================================================================

.PHONY: bundle
bundle: manifests kustomize operator-sdk ## Generate bundle manifests and metadata, then validate generated files.
	"$(OPERATOR_SDK)" generate kustomize manifests -q
	cd config/manager && "$(KUSTOMIZE)" edit set image controller=$(IMG)
	"$(KUSTOMIZE)" build config/manifests | "$(OPERATOR_SDK)" generate bundle $(BUNDLE_GEN_FLAGS)
	"$(OPERATOR_SDK)" bundle validate ./bundle

.PHONY: bundle-build
bundle-build: ## Build the bundle image.
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

.PHONY: bundle-push
bundle-push: ## Push the bundle image.
	$(MAKE) docker-push IMG=$(BUNDLE_IMG)

.PHONY: catalog-build
catalog-build: opm ## Build a catalog image.
	$(OPM) index add --container-tool docker --mode semver --tag $(CATALOG_IMG) --bundles $(BUNDLE_IMGS) $(FROM_INDEX_OPT)

.PHONY: catalog-push
catalog-push: ## Push a catalog image.
	$(MAKE) docker-push IMG=$(CATALOG_IMG)

.PHONY: non-olm
non-olm: kustomize ## Generate non-olm deployment manifest
	"$(KUSTOMIZE)" build config/non-olm -o deploy/falcon-operator.yaml
