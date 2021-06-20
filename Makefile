
VERSION ?= latest
# Image URL to use all building/pushing image targets
IMG ?= ghcr.io/pelotech/jsonnet-controller:$(VERSION)
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true,preserveUnknownFields=false"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

embed-bundle:
	cp config/bundle/manifest.yaml pkg/cmd/manifest.yaml

all: build

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

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

manifests: embed-bundle controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

generate: embed-bundle controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
test: embed-bundle manifests generate fmt vet ## Run tests.
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.8.3/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out

GOLANGCI_LINT    ?= $(PWD)/bin/golangci-lint
GOLANGCI_VERSION ?= v1.40.1
$(GOLANGCI_LINT):
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b "$(PWD)/bin" $(GOLANGCI_VERSION)

lint: $(GOLANGCI_LINT) embed-bundle ## Run linting.
	"$(GOLANGCI_LINT)" run -v

##@ Build

build: generate fmt vet ## Build manager binary.
	go build -o bin/manager main.go

run: manifests generate fmt vet ## Run a controller from your host.
	go run ./main.go

docker-build: test embed-bundle ## Build docker image with the manager.
	docker build -t ${IMG} .

docker-push: ## Push docker image with the manager.
	docker push ${IMG}

REFDOCS = $(CURDIR)/bin/refdocs
$(REFDOCS):
	cd hack/gen-crd-reference-docs && go build -o $(REFDOCS) .

api-docs: $(REFDOCS)  ## Generate API documentation
	go mod vendor
	bash hack/update-api-docs.sh

bundle: ## Generate the bundle manifest
	$(KUBECFG) show --tla-str version=$(VERSION) \
		config/jsonnet/jsonnet-controller.jsonnet > config/bundle/manifest.yaml

##@ Deployment

install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.1)

KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v3@v3.8.7)

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

license-headers:
	for i in `find . -type f \
		-not -wholename '.git/*' \
		-not -wholename './vendor/*' \
		-name '*.go'` ; do \
			if ! grep -q Copyright $$i ; then cat hack/boilerplate.go.txt $$i > $$i.new && mv $$i.new $$i ; fi ; \
	done

##@ Testing in k3d

KUBECTL  ?= kubectl
K8S_VER  ?= v1.21.1
BIN_DIR  ?= $(CURDIR)/bin

K3D         ?= bin/k3d
K3D_VERSION ?= v4.4.6
$(K3D):
	mkdir -p "$(BIN_DIR)"
	curl -s https://raw.githubusercontent.com/rancher/k3d/main/install.sh | \
		USE_SUDO=false TAG=$(K3D_VERSION) K3D_INSTALL_DIR="$(BIN_DIR)" bash

FLUX ?= bin/flux
$(FLUX):
	mkdir -p "$(BIN_DIR)"
	curl -s https://fluxcd.io/install.sh | \
		bash -s -- "$(BIN_DIR)"

KUBECFG         ?= bin/kubecfg
KUBECFG_VERSION ?= v0.20.0
$(KUBECFG):
	mkdir -p "$(BIN_DIR)"
	curl -JL -o "$(KUBECFG)" https://github.com/bitnami/kubecfg/releases/download/$(KUBECFG_VERSION)/kubecfg-$(shell uname | tr A-Z a-z)-amd64
	chmod +x "$(KUBECFG)"

K3S_IMG      ?= rancher/k3s:$(K8S_VER)-k3s1
CONTEXT      ?= k3d-$(CLUSTER_NAME)
CLUSTER_NAME ?= jsonnet-controller
K3D_CLUSTER_ARGS ?= 

cluster: $(K3D) ## Create a local cluster with k3d
	$(K3D) $(K3D_CLUSTER_ARGS) --image $(K3S_IMG) cluster create $(CLUSTER_NAME)

SOURCE_VER   ?= v0.15.1

flux-crds: ## Install the flux source-controller CRDs to the k3d cluster.
	$(KUBECTL) apply --context=$(CONTEXT) \
		-f https://raw.githubusercontent.com/fluxcd/source-controller/$(SOURCE_VER)/config/crd/bases/source.toolkit.fluxcd.io_gitrepositories.yaml \
	 	-f https://raw.githubusercontent.com/fluxcd/source-controller/$(SOURCE_VER)/config/crd/bases/source.toolkit.fluxcd.io_buckets.yaml

flux-install: $(FLUX) ## Install flux and all its components to the k3d cluster.
	$(FLUX) --context=$(CONTEXT) check --pre
	$(FLUX) --context=$(CONTEXT) install 

docker-load: $(K3D) docker-build ## Load the manager image into the k3d cluster.
	$(K3D) image import --cluster $(CLUSTER_NAME) $(IMG)

deploy: $(KUBECFG) ## Deploy the manager and CRDs into the k3d cluster.
	$(KUBECFG) --context=$(CONTEXT) --tla-str version=$(VERSION) \
		update config/jsonnet/jsonnet-controller.jsonnet

restart:
	$(KUBECTL) delete --context=$(CONTEXT) \
		-n flux-system -l app=jsonnet-controller pod 

samples: ## Deploy the sample source-controller manifests into the cluster.
	$(KUBECTL) apply --context=$(CONTEXT) \
		-f config/samples/jsonnet-controller-git-repository.yaml \
		-f config/samples/whoami-source-controller-konfiguration.yaml

full-local-env: cluster flux-install docker-load deploy samples ## Creates a full local environment (cluster, flux-full-install, docker-load, deploy, samples).

delete-cluster: $(K3D) ## Delete the k3d cluster.
	$(K3D) cluster delete $(CLUSTER_NAME)

##@ CLI

LDFLAGS ?= -s -w

build-konfig: embed-bundle ## Build the CLI to your GOBIN
	cd cmd/konfig && \
		CGO_ENABLED=0 go build -ldflags="$(LDFLAGS)" -o $(GOBIN)/konfig .

GOX ?= $(GOBIN)/gox
$(GOX):
	GO111MODULE=off go get github.com/mitchellh/gox

DIST ?= $(CURDIR)/dist
COMPILE_TARGETS ?= "darwin/amd64 linux/amd64 linux/arm linux/arm64 windows/amd64"
COMPILE_OUTPUT  ?= "$(DIST)/{{.Dir}}_{{.OS}}_{{.Arch}}"
dist-konfig: embed-bundle $(GOX)  ## Build release artifacts for the CLI
	mkdir -p dist
	cd cmd/konfig && \
		CGO_ENABLED=0 $(GOX) -osarch=$(COMPILE_TARGETS) -output=$(COMPILE_OUTPUT) -ldflags="$(LDFLAGS)"
	upx -9 $(DIST)/*