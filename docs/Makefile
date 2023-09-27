# Kubernetes distros to include in the documentation.
DISTROS = "azure eks gke eks-fargate generic openshift"

# Create custom resource docs to include in the documentation for a specific k8s distro.
RESOURCE_DOCS = "openshift"

# Documentation source directory.
SOURCE_DIR ?= src/

# Documentation build destination directory.
BUILD_DESTINATION ?= ./

# Documentation test build directory.
TEST_DIR ?= testdir

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: docs

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: test
test:
	@mkdir -p $(TEST_DIR)
	go run docgen.go -src=$(SOURCE_DIR) -out=$(TEST_DIR) -include-resource-docs=$(RESOURCE_DOCS) -distros=$(DISTROS)
	git diff --color --no-index $(TEST_DIR)/resources $(BUILD_DESTINATION)/resources
	git diff --color --no-index $(TEST_DIR)/deployment $(BUILD_DESTINATION)/deployment

.PHONY: docs
docs: ## Build the documentation using docgen.go.
	@echo "Building documentation..."
	go run docgen.go -src=$(SOURCE_DIR) -out=$(BUILD_DESTINATION) -include-resource-docs=$(RESOURCE_DOCS) -distros=$(DISTROS)
