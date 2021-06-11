
VERSION ?= latest
# Image URL to use all building/pushing image targets
IMG ?= ghcr.io/pelotech/kubecfg-controller:$(VERSION)
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

manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
test: manifests generate fmt vet ## Run tests.
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.8.3/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out

##@ Build

build: generate fmt vet ## Build manager binary.
	go build -o bin/manager main.go

run: manifests generate fmt vet ## Run a controller from your host.
	go run ./main.go

docker-build: test ## Build docker image with the manager.
	docker build -t ${IMG} .

docker-push: ## Push docker image with the manager.
	docker push ${IMG}

##@ Deployment

install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

# deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
# 	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
# 	$(KUSTOMIZE) build config/default | kubectl apply -f -

# undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
# 	$(KUSTOMIZE) build config/default | kubectl delete -f -


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


## BEGIN CUSTOM TARGETS ##


license-headers:
	for i in `find . -type f \
		-not -wholename '.git/*' \
		-name '*.go'` ; do \
			if ! grep -q Copyright $$i ; then cat hack/boilerplate.go.txt $$i > $$i.new && mv $$i.new $$i ; fi ; \
	done

##@ Testing in k3d

K3D          ?= k3d
KUBECTL      ?= kubectl
KUBECFG      ?= kubecfg
FLUX         ?= flux
CLUSTER_NAME ?= kubecfg-operator
K8S_VER      ?= v1.20.6
SOURCE_VER   ?= v0.14.0
K3S_IMG      ?= rancher/k3s:$(K8S_VER)-k3s1
CONTEXT      ?= k3d-$(CLUSTER_NAME)

K3D_CLUSTER_ARGS       ?=
# Comment this out if you want to. On Linux kernels > 5.11 there is an issue with k3s kubelet and configuring nf_conntrack.
# https://k3d.io/faq/faq/#nodes-fail-to-start-or-get-stuck-in-notready-state-with-log-nf_conntrack_max-permission-denied
K3D_CONNTRACK_FIX_ARGS = --k3s-server-arg --kube-proxy-arg=conntrack-max-per-core=0 --k3s-agent-arg --kube-proxy-arg=conntrack-max-per-core=0

cluster: ## Create a local cluster with k3d
	$(K3D) $(K3D_CONNTRACK_FIX_ARGS) $(K3D_CLUSTER_ARGS) \
		--image $(K3S_IMG) \
		cluster create $(CLUSTER_NAME)

flux-crds: ## Install the flux source-controller CRDs to the k3d cluster.
	$(KUBECTL) apply --context=$(CONTEXT) \
		-f https://raw.githubusercontent.com/fluxcd/source-controller/$(SOURCE_VER)/config/crd/bases/source.toolkit.fluxcd.io_gitrepositories.yaml \
	 	-f https://raw.githubusercontent.com/fluxcd/source-controller/$(SOURCE_VER)/config/crd/bases/source.toolkit.fluxcd.io_buckets.yaml

flux-full-install: ## Install flux and all its components to the k3d cluster.
	$(FLUX) --context=$(CONTEXT) check --pre
	$(FLUX) --context=$(CONTEXT) install 

docker-load: docker-build ## Load the manager image into the k3d cluster.
	$(K3D) image import --cluster $(CLUSTER_NAME) $(IMG)

deploy: ## Deploy the manager and CRDs into the k3d cluster.
	$(KUBECFG) --context=$(CONTEXT) update config/jsonnet/kubecfg-operator.jsonnet

delete-cluster: ## Delete the k3d cluster.
	$(K3D) cluster delete $(CLUSTER_NAME)