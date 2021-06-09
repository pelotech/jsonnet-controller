# kubecfg-operator
An operator for managing remote manifests via kubecfg

---

This project is in very early stages proof-of-concept. There are no images published and to
test it out you will have to build (and/or publish) images yourself.

The ultimate goal is to potentially integrate this project with the Flux [GitOps Toolkit APIs](https://fluxcd.io/docs/gitops-toolkit/), along
with the existing functionality for absolute URLs.

## Development

### Building

You can use the `Makefile` to perform any build operations:

```bash
# After code changes to the API make sure you run deep-copy code and manifest
# generation
make generate manifests

# Builds the docker image
make docker-build

# Builds the docker image with a custom tag
make docker-build IMG=my.repo.com/kubecfg-controller:latest

# Push the docker image (also accepts the IMG argument)
make docker-push
```

### Installation/Testing

The instructions below assume you are using [`k3d`](https://k3d.io) for running a local kubernetes cluster. The instructions will work mostly the same for `kind`, `minikube`, etc. as well.

The most accurate installation manifest is the [jsonnet](config/jsonnet/kubecfg-operator.jsonnet) file. 
You may also use the `kubebuilder` generated Kustomize manifests, but you will need to bind `cluster-admin` privileges to the manager yourself.

To use the `jsonnet` you will need to install [`kubecfg`](https://github.com/bitnami/kubecfg/releases).

```bash
# Make a test cluster
k3d cluster create

# Import the built image into the cluster if you did not push it
# to a repository. Replace the image name with any overrides you did.
k3d image import ghcr.io/pelotech/kubecfg-controller:latest

# Deploy the manager and CRDs to the cluster using kubecfg.
kubecfg update config/jsonnet/kubecfg-operator.jsonnet

# To deploy the manager with support for flux's source-controller, run the 'overlay'
# instead:
kubecfg update config/jsonnet/kubecfg-operator-flux.jsonnet
```

There is a very-simple example of a `Konfiguration` manifest [here](config/samples/whoami.yaml).
It uses the simple `jsonnet` [whoami-example](config/jsonnet/whoami.jsonnet) included in this repo.
You can apply it with `kubectl`.

```bash
kubectl apply -f config/samples/apps_v1_konfiguration.yaml
```

There are also examples of using this controller with Flux's `source-controller` now.
But, like the rest of this project, this is all very PoC still. 
The examples use the whoami jsonnet snippets in this repository as well.
See the example [GitRepository](hack/manifests/git-repo.yaml) and [Konfiguration](hack/manifests/konfig.yaml).

---

There will be generated documentation later, but for now to see all Konfiguration options, view the [source code](api/v1/konfiguration_types.go) (specifically the `json` tags).