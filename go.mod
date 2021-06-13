module github.com/pelotech/kubecfg-operator

go 1.16

require (
	github.com/cyphar/filepath-securejoin v0.2.2
	github.com/fluxcd/pkg/apis/meta v0.9.0
	github.com/fluxcd/pkg/runtime v0.11.1
	github.com/fluxcd/pkg/untar v0.1.0
	github.com/fluxcd/source-controller/api v0.13.2
	github.com/go-logr/logr v0.4.0
	github.com/hashicorp/go-retryablehttp v0.7.0
	github.com/onsi/ginkgo v1.14.2
	github.com/onsi/gomega v1.10.2
	gopkg.in/yaml.v2 v2.4.0
	k8s.io/api v0.21.1
	k8s.io/apimachinery v0.21.1
	k8s.io/client-go v0.21.1
	sigs.k8s.io/cli-utils v0.25.0
	sigs.k8s.io/controller-runtime v0.8.3
)
