/*
Copyright 2021 Pelotech.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"errors"
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
var clientErr error
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

func checkClient() error {
	if clientErr != nil {
		return clientErr
	}
	if k8sClient == nil {
		return errors.New("could not configure the k8s client")
	}
	return nil
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
			clientErr = err
			return
		}
		kubeconfig = filepath.Join(usr.HomeDir, ".kube", "config")
	}

	kbytes, err := ioutil.ReadFile(kubeconfig)
	if err != nil {
		clientErr = err
		return
	}

	restConfig, err = clientcmd.RESTConfigFromKubeConfig(kbytes)
	if err != nil {
		clientErr = err
		return
	}

	restMapper, err := apiutil.NewDynamicRESTMapper(restConfig)
	if err != nil {
		clientErr = err
		return
	}

	k8sClient, err = client.New(restConfig, client.Options{
		Mapper: restMapper,
		Scheme: scheme,
	})
	if err != nil {
		clientErr = err
		return
	}
}
