# jsonnet-controller

A fluxcd controller for managing manifests declared in jsonnet.

Kubecfg (and its internal libraries) as well as Tanka-style directories with a `main.jsonnet` are supported.

---

This project is in very early stages proof-of-concept. Only `latest` images are published, and they are not guaranteed stable at the moment.

## Quickstart

API Documentation is available [here](doc/konfigurations.md#Konfiguration).

### Installing

You can use either `kubectl`, [`kustomize`](https://kubectl.docs.kubernetes.io/installation/kustomize/binaries/), or [`kubecfg`](https://github.com/bitnami/kubecfg/releases) to install the controller and its CRDs.

```bash
# Using kubectl - A bundle manifest is included in the repository that is
# the output of the jsonnet with the default values.
kubectl apply -f https://github.com/pelotech/jsonnet-controller/raw/main/config/bundle/manifest.yaml

# Using kubecfg - import and extend this file for modifications
## The kubecfg assumes the `flux-system` namespace is present already.
## If it isn't, create it (or import the file and set `create_namespace: true`):
##    kubectl create ns flux-system
kubecfg update config/jsonnet/jsonnet-controller.jsonnet

# Using kustomize
cd config/manager
## This is the current value, but if you want to change the image
kustomize edit set image controller=ghcr.io/pelotech/jsonnet-controller:latest
## Deploy
kustomize build . | kubectl apply -f -
```

Using `kustomize` you will want to tie any additional cluster permissions necessary to the created manager role.
The jsonnet by default makes the manager a cluster-admin.

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
apiVersion: jsonnet.io/v1
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
apiVersion: jsonnet.io/v1
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
- [ ] Metrics
- [ ] Better Validation Options
- [ ] Better Patch Strategies
