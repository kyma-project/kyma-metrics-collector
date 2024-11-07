APP_NAME = kyma-metrics-collector

# Image URL to use all building/pushing image targets
IMG_REGISTRY_PORT ?= 8888
IMG_REGISTRY ?= op-skr-registry.localhost:$(IMG_REGISTRY_PORT)/unsigned/eventing-images
IMG ?= $(IMG_REGISTRY)/$(APP_NAME):$(APP_VERSION)

# Lint issue category
CATEGORY = "default"

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

# VERIFY_IGNORE is a grep pattern to exclude files and directories from verification
VERIFY_IGNORE := /vendor\|/mocks

# FILES_TO_CHECK is a command used to determine which files should be verified
FILES_TO_CHECK = find . -type f -name "*.go" | grep -v "$(VERIFY_IGNORE)"
# DIRS_TO_CHECK is a command used to determine which directories should be verified
DIRS_TO_CHECK = go list ./... | grep -v "$(VERIFY_IGNORE)"
# DIRS_TO_IGNORE is a command used to determine which directories should not be verified
DIRS_TO_IGNORE = go list ./... | grep "$(VERIFY_IGNORE)"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
#
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

SRC_ROOT := $(shell git rev-parse --show-toplevel)
TOOLS_MOD_DIR    := $(SRC_ROOT)/internal/tools
TOOLS_MOD_REGEX  := "\s+_\s+\".*\""
TOOLS_PKG_NAMES  := $(shell grep -E $(TOOLS_MOD_REGEX) < $(TOOLS_MOD_DIR)/tools.go | tr -d " _\"")
TOOLS_BIN_DIR    := $(SRC_ROOT)/bin
# Strip off versions (e.g. /v2) from pkg names
TOOLS_PKG_NAMES_CLEAN  := $(shell grep -E $(TOOLS_MOD_REGEX) < $(TOOLS_MOD_DIR)/tools.go | tr -d " _\"" | sed "s/\/v[0-9].*$$//")
TOOLS_BIN_NAMES  := $(addprefix $(TOOLS_BIN_DIR)/, $(notdir $(TOOLS_PKG_NAMES_CLEAN)))

.PHONY: install-tools
install-tools: $(TOOLS_BIN_NAMES)

$(TOOLS_BIN_DIR):
	mkdir -p $@

$(TOOLS_BIN_NAMES): $(TOOLS_BIN_DIR) $(TOOLS_MOD_DIR)/go.mod
	cd $(TOOLS_MOD_DIR) && go build -o $@ -trimpath $(filter $(filter %/$(notdir $@),$(TOOLS_PKG_NAMES_CLEAN))%,$(TOOLS_PKG_NAMES))

## Tools
GOLANGCI_LINT    := $(TOOLS_BIN_DIR)/golangci-lint
WSL              := $(TOOLS_BIN_DIR)/wsl

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

##@ Development

.PHONY: lint
lint: $(GOLANGCI_LINT) ## Check lint issues using `golangci-lint`
	$(GOLANGCI_LINT) run

.PHONY: lint-compact
lint-compact: $(GOLANGCI_LINT) ## Check lint issues using `golangci-lint` in compact result format
	$(GOLANGCI_LINT) run --print-issued-lines=false

.PHONY: lint-fix
lint-fix: $(GOLANGCI_LINT) $(WSL) ## Check and fix lint issues using `golangci-lint`
	$(WSL) -fix ./...
	$(GOLANGCI_LINT) run --fix

.PHONY: lint-report
lint-report: $(GOLANGCI_LINT)golangci-lint ## Check lint issues using `golangci-lint` then export them to a file, then print the list of linters used
	$(GOLANGCI_LINT) run --issues-exit-code 0 --out-format json > ./lint-report.json

.PHONY: lint-report-issue-category
lint-report-issue-category: ## Get lint issues categories
	make lint-report-clean
	make lint-report
	cat ./lint-report.json | jq '.Issues[].FromLinter' | jq -s 'map({(.):1})|add|keys_unsorted'

.PHONY: lint-report-get-category
lint-report-get-category: ## Get lint issues by category
	cat ./lint-report.json | jq --arg CATEGORY $$CATEGORY '.Issues[] | select(.FromLinter==$$CATEGORY)'

.PHONY: lint-report-count-issue
lint-report-count-issue: ## Count lint issues
	make lint-report-clean
	make lint-report
	cat ./lint-report.json | jq '.Issues | length'

.PHONY: lint-report-clean
lint-report-clean: ## Clean lint report
	rm -f ./lint-report.json

.PHONY: fmt
fmt: ## Reformat files using `go fmt`
	go fmt $$($(DIRS_TO_CHECK))

.PHONY: imports
imports: ## Optimize imports
	goimports -w -l $$($(FILES_TO_CHECK))

.PHONY: go-gen
go-gen:
	go generate ./...

.PHONY: vendor
vendor:
	go mod vendor

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: generate-and-test
generate-and-test: vendor go-gen fmt imports vet lint test; ## Generate and run tests

.PHONY: clean
clean: ## Clean the vendor directory
	rm -rf vendor

test: ## Run tests
	mkdir -p /tmp/artifacts
	go test -coverprofile=/tmp/artifacts/cover.out ./...
	@echo -n "Total coverage: "
	@go tool cover -func=/tmp/artifacts/cover.out | grep total | awk '{print $$3}'

update-metrics-docs: ## Updates the metrics document (i.e. metrics.md).
	@metrics=$$(mktemp).json; echo $${metrics}; promlinter list -ojson . > $${metrics}; gomplate -d kmc=$${metrics} -f hack/metrics.md.tpl  | prettier --parser markdown > metrics.md


##@ Build

.PHONY: build
build: go-gen fmt vet ## Build KMC binary.
	go build -o bin/kyma-metrics-collector cmd/main.go

.PHONY: run
run: go-gen fmt vet ## Run KMC from your host.
	go run ./cmd/main.go

.PHONY: release-image
release-image: docker-build docker-push ## Build the KMC Image and push it to a registry defined in IMG_REGISTRY
	echo "built and pushed KMC image $(IMG)"

# If you wish built the manager image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64 ). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/develop-images/build_enhancements/
.PHONY: docker-build
docker-build: ## Build docker image with the manager.
	docker build -t ${IMG} .

.PHONY: docker-push
docker-push: ## Push docker image with the manager.
	docker push ${IMG}

# PLATFORMS defines the target platforms for  the manager image be build to provide support to multiple
# architectures. (i.e. make docker-buildx IMG=myregistry/mypoperator:0.0.1). To use this option you need to:
# - able to use docker buildx . More info: https://docs.docker.com/build/buildx/
# - have enable BuildKit, More info: https://docs.docker.com/develop/develop-images/build_enhancements/
# - be able to push the image for your registry (i.e. if you do not inform a valid value via IMG=<myregistry/image:<tag>> then the export will fail)
# To properly provided solutions that supports more than one platform you should use this option.
PLATFORMS ?= linux/arm64,linux/amd64,linux/s390x,linux/ppc64le
.PHONY: docker-buildx
docker-buildx: test ## Build and push docker image for the KMC for cross-platform support
	# copy existing Dockerfile and insert --platform=${BUILDPLATFORM} into Dockerfile.cross, and preserve the original Dockerfile
	sed -e '1 s/\(^FROM\)/FROM --platform=\$$\{BUILDPLATFORM\}/; t' -e ' 1,// s//FROM --platform=\$$\{BUILDPLATFORM\}/' Dockerfile > Dockerfile.cross
	- docker buildx create --name project-v3-builder
	docker buildx use project-v3-builder
	- docker buildx build --push --platform=$(PLATFORMS) --tag ${IMG} -f Dockerfile.cross .
	- docker buildx rm project-v3-builder
	rm Dockerfile.cross
