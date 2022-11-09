module github.com/pelotech/jsonnet-controller

go 1.16

require (
	github.com/Masterminds/sprig/v3 v3.2.2
	github.com/cyphar/filepath-securejoin v0.2.3
	github.com/davecgh/go-spew v1.1.1
	github.com/evanphx/json-patch v4.11.0+incompatible
	github.com/fluxcd/pkg/apis/meta v0.10.1
	github.com/fluxcd/pkg/runtime v0.12.1
	github.com/fluxcd/pkg/untar v0.1.0
	github.com/fluxcd/source-controller/api v0.15.4
	github.com/go-logr/logr v1.2.3
	github.com/google/go-jsonnet v0.17.0
	github.com/hashicorp/go-retryablehttp v0.7.0
	github.com/onsi/ginkgo v1.16.4
	github.com/onsi/gomega v1.24.1
	github.com/pkg/errors v0.9.1
	github.com/russross/blackfriday/v2 v2.1.0
	github.com/spf13/cobra v1.2.1
	gopkg.in/yaml.v2 v2.4.0
	gopkg.in/yaml.v3 v3.0.1
	helm.sh/helm/v3 v3.6.3
	k8s.io/api v0.21.3
	k8s.io/apiextensions-apiserver v0.21.3
	k8s.io/apimachinery v0.21.3
	k8s.io/client-go v0.21.3
	k8s.io/gengo v0.0.0-20210203185629-de9496dff47b
	k8s.io/klog v1.0.0
	sigs.k8s.io/cli-utils v0.25.1-0.20210608181808-f3974341173a
	sigs.k8s.io/controller-runtime v0.9.5
)
