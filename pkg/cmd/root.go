package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"

	"github.com/spf13/cobra"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"

	konfigurationv1 "github.com/pelotech/jsonnet-controller/api/v1beta1"
)

func init() {
	cobra.OnInitialize(initClient)

	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "The path to a kubernetes kubeconfig, defaults to ~/.kube/config.")
}

var kubeconfig string

var restConfig *rest.Config
var k8sClient client.Client
var serializer *json.Serializer

var rootCmd = &cobra.Command{
	Use:          "konfig",
	Short:        "Simple CLI utility for jsonnet-controller operations",
	SilenceUsage: true,
}

// Execute executes the cobra command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func initClient() {
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(konfigurationv1.AddToScheme(scheme))

	serializer = json.NewSerializerWithOptions(json.DefaultMetaFactory, scheme, scheme, json.SerializerOptions{
		Pretty: true,
		Yaml:   true,
		Strict: true,
	})

	if kubeconfig == "" {
		usr, err := user.Current()
		if err != nil {
			fmt.Fprint(os.Stderr, "ERROR: Fatal could not determine current user's home directory:", err.Error())
			os.Exit(1)
		}
		kubeconfig = filepath.Join(usr.HomeDir, ".kube", "config")
	}

	kbytes, err := ioutil.ReadFile(kubeconfig)
	if err != nil {
		fmt.Fprint(os.Stderr, "Could not read", kubeconfig, ":", err.Error())
		os.Exit(1)
	}

	restConfig, err = clientcmd.RESTConfigFromKubeConfig(kbytes)
	if err != nil {
		fmt.Fprint(os.Stderr, "Could not create a REST config from", kubeconfig, ":", err.Error())
		os.Exit(1)
	}

	restMapper, err := apiutil.NewDynamicRESTMapper(restConfig)
	if err != nil {
		fmt.Fprint(os.Stderr, "Could not setup a REST Mapper from", kubeconfig, ":", err.Error())
		os.Exit(1)
	}

	k8sClient, err = client.New(restConfig, client.Options{
		Mapper: restMapper,
		Scheme: scheme,
	})
	if err != nil {
		fmt.Fprint(os.Stderr, "Failed to configure a Kubernetes client:", err.Error())
		os.Exit(1)
	}
}
