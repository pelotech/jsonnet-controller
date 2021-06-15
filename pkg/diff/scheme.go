package diff

import (
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"

	sourcev1 "github.com/fluxcd/source-controller/api/v1beta1"
	konfigurationv1 "github.com/pelotech/jsonnet-controller/api/v1"
)

var Scheme = runtime.NewScheme()

func init() {
	utilruntime.Must(clientgoscheme.AddToScheme(Scheme))
	utilruntime.Must(konfigurationv1.AddToScheme(Scheme))
	utilruntime.Must(sourcev1.AddToScheme(Scheme))
}
