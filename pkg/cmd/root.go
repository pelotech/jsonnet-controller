package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/user"
	"path/filepath"

	"github.com/spf13/cobra"

	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/apiutil"
)

func init() {
	cobra.OnInitialize(initClient)

	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "The path to a kubernetes kubeconfig, defaults to ~/.kube/config.")
}

var kubeconfig string

var restConfig *rest.Config
var k8sClient client.Client

var rootCmd = &cobra.Command{
	Use:          "jctl",
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
	})
	if err != nil {
		fmt.Fprint(os.Stderr, "Failed to configure a Kubernetes client:", err.Error())
		os.Exit(1)
	}
}
