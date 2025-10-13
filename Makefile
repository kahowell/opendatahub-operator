# VERSION defines the project version for the bundle.
# Update this value when you upgrade the version of your project.
# To re-generate a bundle for another specific version without changing the standard setup, you can:
# - use the VERSION as arg of the bundle target (e.g make bundle VERSION=0.0.2)
# - use environment variables to overwrite this value (e.g export VERSION=0.0.2)
VERSION ?= 3.0.0
# IMAGE_TAG_BASE defines the opendatahub.io namespace and part of the image name for remote images.
# This variable is used to construct full image tags for bundle and catalog images.
#
# For example, running 'make bundle-build bundle-push catalog-build catalog-push' will build and push both
# opendatahub.io/opendatahub-operator-bundle:$VERSION and opendatahub.io/opendatahub-operator-catalog:$VERSION.
IMAGE_TAG_BASE ?= quay.io/opendatahub/opendatahub-operator

# keep the name based on IMG which already used from command line
IMG_TAG ?= latest
# Update IMG to a variable, to keep it consistent across versions for OpenShift CI
IMG ?= $(IMAGE_TAG_BASE):$(IMG_TAG)
# BUNDLE_IMG defines the image:tag used for the bundle.
# You can use it as an arg. (E.g make bundle-build BUNDLE_IMG=<some-registry>/<project-name-bundle>:<tag>)
BUNDLE_IMG ?= $(IMAGE_TAG_BASE)-bundle:v$(VERSION)

IMAGE_BUILDER ?= podman
OPERATOR_NAMESPACE ?= opendatahub-operator-system
DEFAULT_MANIFESTS_PATH ?= opt/manifests
CGO_ENABLED ?= 1
USE_LOCAL = false

# BUNDLE_CHANNELS defines the bundle channel used in the bundle
BUNDLE_CHANNELS := --channels=fast

# BUNDLE_GEN_FLAGS are the flags passed to the operator-sdk generate bundle command
BUNDLE_GEN_FLAGS ?= -q --overwrite --version $(VERSION) $(BUNDLE_CHANNELS)

# USE_IMAGE_DIGESTS defines if images are resolved via tags or digests
# You can enable this value if you would like to use SHA Based Digests
# To enable set flag to true
USE_IMAGE_DIGESTS ?= false
ifeq ($(USE_IMAGE_DIGESTS), true)
	BUNDLE_GEN_FLAGS += --use-image-digests
endif

##@ Build Dependencies

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
OPERATOR_SDK ?= $(LOCALBIN)/operator-sdk
GOLANGCI_LINT ?= $(LOCALBIN)/golangci-lint
CRD_REF_DOCS ?= $(LOCALBIN)/crd-ref-docs
GINKGO ?= $(LOCALBIN)/ginkgo
YQ ?= $(LOCALBIN)/yq

## Tool Versions
KUSTOMIZE_VERSION ?= v5.7.0
CONTROLLER_TOOLS_VERSION ?= v0.17.3
OPERATOR_SDK_VERSION ?= v1.39.2
GOLANGCI_LINT_VERSION ?= v2.5.0
YQ_VERSION ?= v4.12.2
#ENVTEST_K8S_VERSION is the version of Kubernetes to use for setting up ENVTEST binaries (i.e. 1.31)
ENVTEST_K8S_VERSION ?= $(shell go list -m -f "{{ .Version }}" k8s.io/api | awk -F'[v.]' '{printf "1.%d", $$3}')
#ENVTEST_VERSION is the version of controller-runtime release branch to fetch the envtest setup script (i.e. release-0.20)
ENVTEST_VERSION ?= $(shell go list -m -f "{{ .Version }}" sigs.k8s.io/controller-runtime | awk -F'[v.]' '{printf "release-%d.%d", $$2, $$3}')
CRD_REF_DOCS_VERSION = 0.2.0
# Add to tool versions section
GINKGO_VERSION ?= v2.23.4


PLATFORM ?= linux/amd64

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

# E2E tests additional flags
# See README.md, default go test timeout 10m
E2E_TEST_FLAGS = -timeout 50m

# Default image-build is to not use local odh-manifests folder
# set to "true" to use local instead
# see target "image-build"
IMAGE_BUILD_FLAGS ?= --build-arg USE_LOCAL=$(USE_LOCAL)
IMAGE_BUILD_FLAGS += --build-arg CGO_ENABLED=$(CGO_ENABLED)
IMAGE_BUILD_FLAGS += --platform $(PLATFORM)

# Prometheus-Unit Tests Parameters
PROMETHEUS_CONFIG_YAML = ./config/monitoring/prometheus/apps/prometheus-configs.yaml
PROMETHEUS_CONFIG_DIR = ./config/monitoring/prometheus/apps
PROMETHEUS_TEST_DIR = ./tests/prometheus_unit_tests
PROMETHEUS_ALERT_TESTS = $(wildcard $(PROMETHEUS_TEST_DIR)/*.unit-tests.yaml)

ALERT_SEVERITY = critical

# Read any custom variables overrides from a local.mk file.  This will only be read if it exists in the
# same directory as this Makefile.  Variables can be specified in the standard format supported by
# GNU Make since `include` processes any valid Makefile
# Standard variables override would include anything you would pass at runtime that is different
# from the defaults specified in this file
OPERATOR_MAKE_ENV_FILE = local.mk
-include $(OPERATOR_MAKE_ENV_FILE)

.PHONY: default
default: manifests generate lint unit-test build

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

define go-mod-version
$(shell go mod graph | grep $(1) | head -n 1 | cut -d'@' -f 2)
endef

# Using controller-gen to fetch external CRDs and put them in config/crd/external folder
# They're used in tests, as they have to be created for controller to work
define fetch-external-crds
GOFLAGS="-mod=readonly" $(CONTROLLER_GEN) crd \
paths=$(shell go env GOPATH)/pkg/mod/$(1)@$(call go-mod-version,$(1))/$(2)/... \
output:crd:artifacts:config=config/crd/external
endef

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=controller-manager-role crd:ignoreUnexportedFields=true webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	$(call fetch-external-crds,github.com/openshift/api,route/v1)
	$(call fetch-external-crds,github.com/openshift/api,user/v1)
	$(call fetch-external-crds,github.com/openshift/api,config/v1)
	$(call fetch-external-crds,github.com/openshift/api,operator/v1)
	$(call fetch-external-crds,github.com/openshift/api,template/v1)
	$(call fetch-external-crds,github.com/openshift/api,security/v1)
	$(call fetch-external-crds,github.com/openshift/api,console/v1)

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

GOLANGCI_TMP_FILE = .golangci.mktmp.yml
.PHONY: fmt
fmt: golangci-lint yq ## Formats code and imports.
	go fmt ./...
	$(GOLANGCI_LINT) fmt
CLEANFILES += $(GOLANGCI_TMP_FILE)

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

GOLANGCI_LINT_TIMEOUT ?= 5m0s
.PHONY: lint
lint: golangci-lint ## Run golangci-lint against code.
	$(GOLANGCI_LINT) run --timeout=$(GOLANGCI_LINT_TIMEOUT)

.PHONY: lint-fix
lint-fix: golangci-lint ## Run golangci-lint against code.
	$(GOLANGCI_LINT) run --fix
	$(GOLANGCI_LINT) fmt

.PHONY: get-manifests
get-manifests: ## Fetch components manifests from remote git repo
	./get_all_manifests.sh
CLEANFILES += opt/manifests/*

# Default to standard sed command
SED_COMMAND = sed

# macOS requires GNU sed due to BSD sed syntax differences
ifeq ($(shell uname -s),Darwin)
    # Verify gsed is available, fail with a helpful message if not installed
    ifeq ($(shell which gsed),)
        $(error gsed not found. Install with: brew install gnu-sed)
    endif
    SED_COMMAND = gsed
endif
.PHONY: api-docs
api-docs: crd-ref-docs ## Creates API docs using https://github.com/elastic/crd-ref-docs, render managementstate with marker
	$(CRD_REF_DOCS) --source-path ./ --output-path ./docs/api-overview.md --renderer markdown --config ./crd-ref-docs.config.yaml && \
	grep -Ev '\.io/[^v][^1].*)$$' ./docs/api-overview.md > temp.md && mv ./temp.md ./docs/api-overview.md && \
	$(SED_COMMAND) -i "s|](#managementstate)|](https://pkg.go.dev/github.com/openshift/api@v0.0.0-20250812222054-88b2b21555f3/operator/v1#ManagementState)|g" ./docs/api-overview.md

.PHONY: ginkgo
ginkgo: $(GINKGO)
$(GINKGO): $(LOCALBIN)
	$(call go-install-tool,$(GINKGO),github.com/onsi/ginkgo/v2/ginkgo,$(GINKGO_VERSION))

##@ Build

.PHONY: build
build: manifests generate fmt vet ## Build manager binary.
	go build -o bin/manager cmd/main.go

RUN_ARGS = --log-mode=devel --pprof-bind-address=127.0.0.1:6060
GO_RUN_MAIN = OPERATOR_NAMESPACE=$(OPERATOR_NAMESPACE) DEFAULT_MANIFESTS_PATH=$(DEFAULT_MANIFESTS_PATH) go run $(GO_RUN_ARGS) ./cmd/main.go $(RUN_ARGS)
.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host.
	$(GO_RUN_MAIN)

.PHONY: run-nowebhook
run-nowebhook: GO_RUN_ARGS += -tags nowebhook

run-nowebhook: manifests generate fmt vet ## Run a controller from your host without webhook enabled
	$(GO_RUN_MAIN)


.PHONY: image-build
image-build: # unit-test ## Build image with the manager.
	$(IMAGE_BUILDER) buildx build --no-cache -f Dockerfiles/Dockerfile ${IMAGE_BUILD_FLAGS} -t $(IMG) .

.PHONY: image-push
image-push: ## Push image with the manager.
	$(IMAGE_BUILDER) push $(IMG)

.PHONY: image
image: image-build image-push ## Build and push image with the manager.

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: prepare
prepare: manifests kustomize manager-kustomization

# phony target for the case of changing IMG variable
.PHONY: manager-kustomization
manager-kustomization: config/manager/kustomization.yaml.in
	cd config/manager \
		&& cp -f kustomization.yaml.in kustomization.yaml \
		&& $(KUSTOMIZE) edit set image controller=$(IMG)

.PHONY: install
install: prepare ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: prepare ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: prepare ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build --load-restrictor LoadRestrictionsNone config/default | kubectl apply --namespace $(OPERATOR_NAMESPACE) -f -

.PHONY: undeploy
undeploy: prepare ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build --load-restrictor LoadRestrictionsNone config/default | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)
CLEANFILES += $(LOCALBIN)

KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	$(call go-install-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v5,$(KUSTOMIZE_VERSION))

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen,$(CONTROLLER_TOOLS_VERSION))

.PHONY: yq
yq: $(YQ) ## Download yq locally if necessary.
$(YQ): $(LOCALBIN)
	$(call go-install-tool,$(YQ),github.com/mikefarah/yq/v4,$(YQ_VERSION))

OPERATOR_SDK_DL_URL ?= https://github.com/operator-framework/operator-sdk/releases/download/$(OPERATOR_SDK_VERSION)
.PHONY: operator-sdk
operator-sdk: $(OPERATOR_SDK) ## Download and install operator-sdk
$(OPERATOR_SDK): $(LOCALBIN)
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	test -s $(OPERATOR_SDK) || curl -sSLo $(OPERATOR_SDK) $(OPERATOR_SDK_DL_URL)/operator-sdk_$${OS}_$${ARCH} && \
	chmod +x $(OPERATOR_SDK) ;\

.PHONY: golangci-lint
golangci-lint: $(GOLANGCI_LINT) ## Download golangci-lint locally if necessary.
$(GOLANGCI_LINT): $(LOCALBIN)
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/v2/cmd/golangci-lint,$(GOLANGCI_LINT_VERSION))

OS=$(shell uname -s)
ARCH=$(shell uname -m)
.PHONY: crd-ref-docs
crd-ref-docs: $(CRD_REF_DOCS)
$(CRD_REF_DOCS): $(LOCALBIN)
	test -s $(CRD_REF_DOCS) || ( \
		curl -sSL https://github.com/elastic/crd-ref-docs/releases/download/v$(CRD_REF_DOCS_VERSION)/crd-ref-docs_$(CRD_REF_DOCS_VERSION)_$(OS)_$(ARCH).tar.gz | tar -xzf - -C $(LOCALBIN) crd-ref-docs \
	)

.PHONY: new-component
new-component: $(LOCALBIN)/component-codegen
	$< generate $(COMPONENT)
	$(MAKE) generate manifests api-docs bundle fmt

$(LOCALBIN)/component-codegen: | $(LOCALBIN)
	cd ./cmd/component-codegen && go mod tidy && go build -o $@

# TODO: change kustomize layout to remove --load-restrictor LoadRestrictionsNone option in each kustomize invocation in this makefile
BUNDLE_DIR ?= "bundle"
WARNINGMSG = "provided API should have an example annotation"
.PHONY: bundle
bundle: prepare operator-sdk ## Generate bundle manifests and metadata, then validate generated files.
	$(OPERATOR_SDK) generate kustomize manifests -q
	$(KUSTOMIZE) build --load-restrictor LoadRestrictionsNone config/manifests | $(OPERATOR_SDK) generate bundle $(BUNDLE_GEN_FLAGS) 2>&1 | grep -v $(WARNINGMSG)
	$(OPERATOR_SDK) bundle validate ./$(BUNDLE_DIR) 2>&1 | grep -v $(WARNINGMSG)
	mv bundle.Dockerfile Dockerfiles/
	rm -f bundle/manifests/opendatahub-operator-webhook-service_v1_service.yaml

.PHONY: bundle-build
bundle-build: bundle
	$(IMAGE_BUILDER) build --no-cache -f Dockerfiles/bundle.Dockerfile --platform $(PLATFORM) -t $(BUNDLE_IMG) .

.PHONY: bundle-push
bundle-push: ## Push the bundle image.
	$(MAKE) image-push IMG=$(BUNDLE_IMG)

.PHONY: deploy-bundle
deploy-bundle: operator-sdk bundle-build bundle-push
	$(OPERATOR_SDK) run bundle $(BUNDLE_IMG)  -n $(OPERATOR_NAMESPACE)

.PHONY: upgrade-bundle
upgrade-bundle: operator-sdk bundle-build bundle-push ## Upgrade bundle
	$(OPERATOR_SDK) run bundle-upgrade $(BUNDLE_IMG) -n $(OPERATOR_NAMESPACE)

.PHONY: opm
OPM = ./bin/opm
opm: ## Download opm locally if necessary.
ifeq (,$(wildcard $(OPM)))
ifeq (,$(shell command -v opm 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPM)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sSLo $(OPM) https://github.com/operator-framework/operator-registry/releases/download/v1.55.0/$${OS}-$${ARCH}-opm ;\
	chmod +x $(OPM) ;\
	}
else
OPM = $(shell which opm)
endif
endif

# A comma-separated list of bundle images (e.g. make catalog-build BUNDLE_IMGS=example.com/operator-bundle:v0.1.0,example.com/operator-bundle:v0.2.0).
# These images MUST exist in a registry and be pull-able.
BUNDLE_IMGS ?= $(BUNDLE_IMG)

# The image tag given to the resulting catalog image (e.g. make catalog-build CATALOG_IMG=example.com/operator-catalog:v0.2.0).
CATALOG_IMG ?= $(IMAGE_TAG_BASE)-catalog:v$(VERSION)

# Set CATALOG_BASE_IMG to an existing catalog image tag to add $BUNDLE_IMGS to that image.
ifneq ($(origin CATALOG_BASE_IMG), undefined)
FROM_INDEX_OPT := --from-index $(CATALOG_BASE_IMG)
endif

.PHONY: catalog-clean
catalog-clean: ## Clean up catalog files and Dockerfile
	rm -rf catalog

.PHONY: catalog-prepare
catalog-prepare: catalog-clean opm yq ## Prepare the catalog by adding bundles to fast channel. It requires BUNDLE_IMG exists before running the target"
	mkdir -p catalog
	cp config/catalog/fbc-basic-template.yaml catalog/fbc-basic-template.yaml
	./hack/update-catalog-template.sh catalog/fbc-basic-template.yaml $(BUNDLE_IMGS)
	$(OPM) alpha render-template basic \
		--migrate-level=bundle-object-to-csv-metadata \
		-o yaml \
		catalog/fbc-basic-template.yaml > catalog/catalog.yaml
	$(OPM) validate catalog
	rm -f catalog/fbc-basic-template.yaml

# Build a catalog image using the operator package manager tool 'opm'.
# This recipe uses 'opm alpha render-template basic' to generate a catalog from a template.
# The template defines bundle images and channel relationships in a declarative way.
.PHONY: catalog-build
catalog-build: catalog-prepare
	$(IMAGE_BUILDER) build --no-cache --load -f Dockerfiles/catalog.Dockerfile --platform $(PLATFORM) -t $(CATALOG_IMG) .

# Push the catalog image.
.PHONY: catalog-push
catalog-push: ## Push a catalog image.
	$(MAKE) image-push IMG=$(CATALOG_IMG)

TOOLBOX_GOLANG_VERSION := 1.24.6

# Generate a Toolbox container for locally testing changes easily
.PHONY: toolbox
toolbox: ## Create a toolbox instance with the proper Golang and Operator SDK versions
	$(IMAGE_BUILDER) build \
		--build-arg GOLANG_VERSION=$(TOOLBOX_GOLANG_VERSION) \
		--build-arg OPERATOR_SDK_VERSION=$(OPERATOR_SDK_VERSION) \
		-f Dockerfiles/toolbox.Dockerfile -t opendatahub-toolbox .
	$(IMAGE_BUILDER) stop opendatahub-toolbox ||:
	toolbox rm opendatahub-toolbox ||:
	toolbox create opendatahub-toolbox --image localhost/opendatahub-toolbox:latest

# Run tests.
TEST_SRC ?=./internal/... ./tests/integration/... ./pkg/...

.PHONY: envtest
envtest: $(ENVTEST) ## Download setup-envtest locally if necessary.
$(ENVTEST): $(LOCALBIN)
	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest,$(ENVTEST_VERSION))

.PHONY: test
test: unit-test e2e-test

.PHONY: unit-test
unit-test: envtest ginkgo # directly use ginkgo since the framework is not compatible with go test parallel
	OPERATOR_NAMESPACE=$(OPERATOR_NAMESPACE) KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" \
    	${GINKGO} -r \
        		--procs=8 \
        		--compilers=2 \
        		--timeout=15m \
        		--poll-progress-after=30s \
        		--poll-progress-interval=5s \
        		--randomize-all \
        		--randomize-suites \
        		--fail-fast \
        		--cover \
        		--coverprofile=cover.out \
        		--succinct \
        		$(TEST_SRC)
CLEANFILES += cover.out

$(PROMETHEUS_TEST_DIR)/%.rules.yaml: $(PROMETHEUS_TEST_DIR)/%.unit-tests.yaml $(PROMETHEUS_CONFIG_YAML) $(YQ)
	$(YQ) eval ".data.\"$(@F:.rules.yaml=.rules)\"" $(PROMETHEUS_CONFIG_YAML) > $@

PROMETHEUS_ALERT_RULES := $(PROMETHEUS_ALERT_TESTS:.unit-tests.yaml=.rules.yaml)

# Run prometheus-alert-unit-tests
.PHONY: test-alerts
test-alerts: $(PROMETHEUS_ALERT_RULES)
	promtool test rules $(PROMETHEUS_ALERT_TESTS)

#Check for alerts without unit-tests
.PHONY: check-prometheus-alert-unit-tests
check-prometheus-alert-unit-tests: $(PROMETHEUS_ALERT_RULES)
	./tests/prometheus_unit_tests/scripts/check_alert_tests.sh $(PROMETHEUS_CONFIG_YAML) $(PROMETHEUS_TEST_DIR) $(ALERT_SEVERITY)
CLEANFILES += $(PROMETHEUS_ALERT_RULES)

.PHONY: e2e-test
e2e-test:
ifndef E2E_TEST_OPERATOR_NAMESPACE
export E2E_TEST_OPERATOR_NAMESPACE = $(OPERATOR_NAMESPACE)
endif
e2e-test: ## Run e2e tests for the controller
	go test ./tests/e2e/ -run ^TestOdhOperator -v ${E2E_TEST_FLAGS}

.PHONY: clean
clean: $(GOLANGCI_LINT)
	$(GOLANGCI_LINT) cache clean
	chmod u+w -R $(LOCALBIN) # envtest makes its dir RO
	rm -rf $(CLEANFILES)

# go-install-tool will 'go install' any package with custom target and name of binary, if it doesn't exist
# $1 - target path with name of binary (ideally with version)
# $2 - package url which can be installed
# $3 - specific version of package
define go-install-tool
@[ -f "$(1)-$(3)" ] || { \
set -e; \
package=$(2)@$(3) ;\
echo "Downloading $${package}" ;\
rm -f $(1) || true ;\
GOBIN=$(LOCALBIN) go install $${package} ;\
mv $(1) $(1)-$(3) ;\
} ;\
ln -sf $(1)-$(3) $(1)
endef

##@ Kind Cluster Management

KIND_CLUSTER_NAME ?= odh-dev
KIND_K8S_VERSION ?= v1.32.0
CERT_MANAGER_VERSION ?= v1.16.2
OLM_VERSION ?= v0.30.0
PROMETHEUS_OPERATOR_VERSION ?= v0.86.0
GATEWAY_API_VERSION ?= v1.4.0

.PHONY: kind-create
kind-create: ## Create a kind cluster for local development
	@echo "Creating kind cluster '$(KIND_CLUSTER_NAME)'..."
	@if kind get clusters | grep -q "^$(KIND_CLUSTER_NAME)$$"; then \
		echo "Cluster '$(KIND_CLUSTER_NAME)' already exists"; \
	else \
		echo 'kind: Cluster' > /tmp/kind-config.yaml; \
		echo 'apiVersion: kind.x-k8s.io/v1alpha4' >> /tmp/kind-config.yaml; \
		echo 'nodes:' >> /tmp/kind-config.yaml; \
		echo '- role: control-plane' >> /tmp/kind-config.yaml; \
		kind create cluster --name $(KIND_CLUSTER_NAME) \
			--image kindest/node:$(KIND_K8S_VERSION) \
			--config /tmp/kind-config.yaml; \
		rm -f /tmp/kind-config.yaml; \
		echo "Cluster '$(KIND_CLUSTER_NAME)' created successfully"; \
	fi
	@kubectl cluster-info --context kind-$(KIND_CLUSTER_NAME)

.PHONY: kind-delete
kind-delete: ## Delete the kind cluster
	@echo "Deleting kind cluster '$(KIND_CLUSTER_NAME)'..."
	@if kind get clusters | grep -q "^$(KIND_CLUSTER_NAME)$$"; then \
		kind delete cluster --name $(KIND_CLUSTER_NAME); \
		echo "Cluster '$(KIND_CLUSTER_NAME)' deleted successfully"; \
	else \
		echo "Cluster '$(KIND_CLUSTER_NAME)' does not exist"; \
	fi

.PHONY: kind-deploy
kind-deploy: prepare image-push install ## Deploy the operator to the kind cluster
	@echo "Deploying operator to kind cluster '$(KIND_CLUSTER_NAME)'..."
	@kubectl config use-context kind-$(KIND_CLUSTER_NAME)
	@echo "Creating operator namespace '$(OPERATOR_NAMESPACE)' if it doesn't exist..."
	@kubectl create namespace $(OPERATOR_NAMESPACE) --dry-run=client -o yaml | kubectl apply -f -
	@echo "Deploying with kind-specific configuration (includes mock cluster version)..."
	$(KUSTOMIZE) build --load-restrictor LoadRestrictionsNone config/kind | kubectl apply --namespace $(OPERATOR_NAMESPACE) -f -
	@echo "Waiting for operator deployment to be ready..."
	@kubectl wait --for=condition=available --timeout=300s \
		deployment/opendatahub-operator-controller-manager -n $(OPERATOR_NAMESPACE) || \
		(echo "Operator deployment failed. Check logs with: kubectl logs -n $(OPERATOR_NAMESPACE) deployment/opendatahub-operator-controller-manager"; exit 1)
	@echo "Operator deployed successfully"

.PHONY: kind-install-cert-manager
kind-install-cert-manager: ## Install cert-manager in the kind cluster
	@echo "Installing cert-manager $(CERT_MANAGER_VERSION)..."
	@if kubectl get namespace cert-manager >/dev/null 2>&1; then \
		echo "cert-manager is already installed"; \
	else \
		kubectl apply -f https://github.com/cert-manager/cert-manager/releases/download/$(CERT_MANAGER_VERSION)/cert-manager.yaml; \
		echo "Waiting for cert-manager to be ready..."; \
		kubectl wait --for=condition=available --timeout=300s deployment/cert-manager -n cert-manager; \
		kubectl wait --for=condition=available --timeout=300s deployment/cert-manager-webhook -n cert-manager; \
		kubectl wait --for=condition=available --timeout=300s deployment/cert-manager-cainjector -n cert-manager; \
		echo "cert-manager installed successfully"; \
	fi

.PHONY: kind-install-olm
kind-install-olm: ## Install OLM (Operator Lifecycle Manager) in the kind cluster
	@echo "Installing OLM $(OLM_VERSION)..."
	@if kubectl get namespace olm >/dev/null 2>&1; then \
		echo "OLM is already installed"; \
	else \
		curl -sL https://github.com/operator-framework/operator-lifecycle-manager/releases/download/$(OLM_VERSION)/install.sh | bash -s $(OLM_VERSION); \
		echo "Waiting for OLM to be ready..."; \
		kubectl wait --for=condition=available --timeout=300s deployment/olm-operator -n olm; \
		kubectl wait --for=condition=available --timeout=300s deployment/catalog-operator -n olm; \
		echo "OLM installed successfully"; \
	fi

.PHONY: kind-install-prometheus-operator
kind-install-prometheus-operator: ## Install Prometheus Operator in the kind cluster
	@echo "Installing Prometheus Operator $(PROMETHEUS_OPERATOR_VERSION)..."
	@if kubectl get namespace default >/dev/null 2>&1 && kubectl get deployment -n default prometheus-operator >/dev/null 2>&1; then \
		echo "Prometheus Operator is already installed"; \
	else \
		kubectl apply --server-side -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/refs/tags/$(PROMETHEUS_OPERATOR_VERSION)/bundle.yaml; \
		echo "Waiting for Prometheus Operator to be ready..."; \
		kubectl wait --for=condition=available --timeout=300s deployment/prometheus-operator -n default; \
		echo "Prometheus Operator installed successfully"; \
	fi

.PHONY: kind-install-gateway-api
kind-install-gateway-api: ## Install Gateway API CRDs in the kind cluster
	@echo "Installing Gateway API $(GATEWAY_API_VERSION)..."
	@if kubectl get crd gateways.gateway.networking.k8s.io >/dev/null 2>&1; then \
		echo "Gateway API is already installed"; \
	else \
		kubectl apply -f https://github.com/kubernetes-sigs/gateway-api/releases/download/$(GATEWAY_API_VERSION)/standard-install.yaml; \
		echo "Gateway API installed successfully"; \
	fi

.PHONY: kind-setup
kind-setup: kind-create kind-install-olm kind-install-cert-manager kind-install-prometheus-operator kind-install-gateway-api image kind-deploy ## Complete kind cluster setup: create cluster, install OLM, cert-manager, prometheus-operator, gateway-api, build and push image, and deploy operator
	@echo "Kind cluster setup complete. You can now run 'make e2e-test'"

.PHONY: kind-status
kind-status: ## Show status of the kind cluster and operator deployment
	@echo "=== Kind Cluster Status ==="
	@if kind get clusters | grep -q "^$(KIND_CLUSTER_NAME)$$"; then \
		echo "Cluster: $(KIND_CLUSTER_NAME) [RUNNING]"; \
		kubectl cluster-info --context kind-$(KIND_CLUSTER_NAME); \
		echo ""; \
		echo "=== Operator Deployment Status ==="; \
		kubectl get deployment -n $(OPERATOR_NAMESPACE) 2>/dev/null || echo "No deployments found in namespace $(OPERATOR_NAMESPACE)"; \
		echo ""; \
		echo "=== Operator Pods ==="; \
		kubectl get pods -n $(OPERATOR_NAMESPACE) 2>/dev/null || echo "No pods found in namespace $(OPERATOR_NAMESPACE)"; \
	else \
		echo "Cluster: $(KIND_CLUSTER_NAME) [NOT FOUND]"; \
		echo "Run 'make kind-create' to create the cluster"; \
	fi

.PHONY: kind-logs
kind-logs: ## Show logs from the operator deployment
	@echo "=== Operator Logs ==="
	@kubectl logs -n $(OPERATOR_NAMESPACE) deployment/opendatahub-operator-controller-manager --tail=100 --follow

.PHONY: kind-restart
kind-restart: ## Restart the operator deployment
	@echo "Restarting operator deployment..."
	@kubectl rollout restart deployment/opendatahub-operator-controller-manager -n $(OPERATOR_NAMESPACE)
	@kubectl rollout status deployment/opendatahub-operator-controller-manager -n $(OPERATOR_NAMESPACE)
