# jsonnet-controller

A fluxcd controller for managing manifests declared in jsonnet.

Kubecfg (and its internal libraries) as well as Tanka-style directories with a `main.jsonnet` are supported.

---

[![Tests](https://github.com/pelotech/jsonnet-controller/actions/workflows/unit_tests.yaml/badge.svg)](https://github.com/pelotech/jsonnet-controller/actions/workflows/unit_tests.yaml)
[![Build](https://github.com/pelotech/jsonnet-controller/actions/workflows/build_images.yaml/badge.svg)](https://github.com/pelotech/jsonnet-controller/actions/workflows/build_images.yaml)
[![Go Report Card](https://goreportcard.com/badge/github.com/pelotech/jsonnet-controller)](https://goreportcard.com/report/github.com/pelotech/jsonnet-controller)

This project is in very early stages proof-of-concept still. So expect bugs. But please feel free to open an Issue if you spot any :smile:.

## Quickstart

API Documentation is available [here](doc/konfigurations.md#konfigurationspec).

### Installing

There are multiple ways to install the `jsonnet-controller`.

#### Using `kubectl`

```bash
VERSION=v0.0.5

kubectl apply -f https://github.com/pelotech/jsonnet-controller/raw/${VERSION}/config/bundle/manifest.yaml
```

You can also use the manifest from the `main` branch to deploy the `latest` tag.

#### Using [`kubecfg`](https://github.com/bitnami/kubecfg/releases)

There is a `kubecfg` manifest located [here](config/jsonnet/jsonnet-controller.jsonnet). You can either invoke it directly, or import it to make your own modifications. You will need to clone the repository first.

```bash
git clone https://github.com/pelotech/jsonnet-controller && cd jsonnet-controller

kubecfg update config/jsonnet/jsonnet-controller.jsonnet
# To install a specific version of the controller
kubecfg update --tla-str version=${VERSION} config/jsonnet/jsonnet-controller.jsonnet
```

#### Using the `konfig` CLI.

There is an experimental CLI included with this project, that among other things, can install the jsonnet-controller into a cluster. The feature is available since `v0.0.5`.
To install, just download the latest CLI from the [releases](https://github.com/pelotech/jsonnet-controller/releases) page and run:

```bash
konfig install
```

Use the `--kubeconfig` flag to specify a kubeconfig different then `~/.kube/config`.

#### Using [`kustomize`](https://kubectl.docs.kubernetes.io/installation/kustomize/binaries/)

Kubebuilder generates `kustomize` manifests with the project. You can use them by cloning the repository down and executing the following:

```bash
git clone https://github.com/pelotech/jsonnet-controller && cd jsonnet-controller

cd config/manager
## This is the current value, but if you want to change the image
kustomize edit set image controller=ghcr.io/pelotech/jsonnet-controller:latest
## Deploy
kustomize build . | kubectl apply -f -
```

Using `kustomize` you will want to tie any additional cluster permissions necessary to the created manager role.
The other installation methods by default make the manager a cluster-admin.

### Examples

First (at the moment this is optional), define a `GitRepository` source for your `Konfiguration`:

```yaml
# config/samples/jsonnet-controller-git-repository.yaml
apiVersion: source.toolkit.fluxcd.io/v1beta1
kind: GitRepository
metadata:
  name: jsonnet-samples
  namespace: flux-system
spec:
  interval: 30s
  ref:
    branch: main
  url: https://github.com/pelotech/jsonnet-controller
```

Finally, create a `Konfiguration` for your application:

```yaml
# config/samples/whoami-source-controller-konfiguration.yaml
apiVersion: jsonnet.io/v1beta1
kind: Konfiguration
metadata:
  name: whoami
spec:
  interval: 30s
  path: config/jsonnet/whoami-tla.jsonnet
  prune: true
  variables:
    tlaStr:
      name: 'whoami'
    tlaCode:
      port: '8080'
  sourceRef:
    kind: GitRepository
    name: jsonnet-samples
    namespace: flux-system
```

This may change, but for now you can choose to skip the `sourceRef` and supply a path to a remote file over HTTP(S).
The file will be checked for changes at the provided interval.

```yaml
apiVersion: jsonnet.io/v1beta1
kind: Konfiguration
metadata:
  name: whoami
spec:
  interval: 30s
  path: https://raw.githubusercontent.com/pelotech/jsonnet-controller/main/config/jsonnet/whoami-tla.jsonnet
  prune: true
  variables:
    tlaStr:
      name: 'whoami'
    tlaCode:
      port: '8080'
```

You can watch the status of the `Konfiguration` with `kubectl`:

```bash
# Available names and shortnames are konfiguration(s), konfig(s), konf(s)
$ kubectl get konfig
NAME     READY   STATUS                                                            AGE
whoami   True    Applied revision: main/0bceb3d69b046f51565a345f3105febbd7be62bd   1m32s

$ kubectl get konfig -o wide
NAME     READY   STATUS                                                            AGE    CURRENTREVISION                                 LASTATTEMPTEDREVISION
whoami   True    Applied revision: main/0bceb3d69b046f51565a345f3105febbd7be62bd   1m38s   main/0bceb3d69b046f51565a345f3105febbd7be62bd   main/0bceb3d69b046f51565a345f3105febbd7be62bd
```

See the [samples](config/samples) directory for more examples.

## Development

### Building

You can use the `Makefile` to perform any build operations:

```bash
# After code changes to the API make sure you run deep-copy code and manifest
# generation
make generate manifests

## Below steps are only if you wish to build your own image. You can also download
## from the public repository.

# Builds the docker image
make docker-build

# Builds the docker image with a custom tag
make docker-build IMG=my.repo.com/jsonnet-controller:latest

# Push the docker image (also accepts the IMG argument)
make docker-push
```

### Local Testing

The instructions below assume you are using [`k3d`](https://k3d.io) for running a local kubernetes cluster. The instructions will work mostly the same for `kind`, `minikube`, etc. as well.

The most accurate installation manifest is the [jsonnet](config/jsonnet/jsonnet-controller.jsonnet) file. 
You may also use the `kubebuilder` generated Kustomize manifests, but you will need to bind `cluster-admin` privileges to the manager yourself.

To use the `jsonnet` you will need to install [`kubecfg`](https://github.com/bitnami/kubecfg/releases).

```bash
# Make a test cluster
k3d cluster create

# Install flux
flux install

# Import the built image into the cluster if you did not push it
# to a repository. Replace the image name with any overrides you did.
# You can skip this step if you wish to pull the image from the public
# repository.
k3d image import ghcr.io/pelotech/jsonnet-controller:latest

# Deploy the manager and CRDs to the cluster using kubecfg.
kubecfg update config/jsonnet/jsonnet-controller.jsonnet
```

There are also `Makefile` helpers to do the equivalent of all of the above:

```bash
make cluster flux-install docker-load deploy
#       |           |          |         |
#   Create Cluster  |          |         |
#              Install Flux    |         |
#                          Load Image    |
#                                 Deploy Controller and CRDs
```

---

## TODO

These are features and other tasks that need to be completed before an initial release will be ready.

- [ ] Unit and E2E Tests
