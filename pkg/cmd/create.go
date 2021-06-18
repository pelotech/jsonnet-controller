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
	"context"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/fluxcd/pkg/apis/meta"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	konfigurationv1 "github.com/pelotech/jsonnet-controller/api/v1beta1"
)

var createSpec = konfigurationv1.Konfiguration{
	TypeMeta: v1.TypeMeta{
		Kind:       "Konfiguration",
		APIVersion: konfigurationv1.GroupVersion.String(),
	},
	ObjectMeta: v1.ObjectMeta{},
	Spec: konfigurationv1.KonfigurationSpec{
		RetryInterval: &v1.Duration{},
		Timeout:       &v1.Duration{},
		Variables:     &konfigurationv1.Variables{},
	},
}
var sourceRef = meta.NamespacedObjectKindReference{}

var konfigKubeConfig string
var injectFile string
var createExport bool

func init() {
	createFlags := createCmd.Flags()

	createFlags.StringVarP(&createSpec.Spec.Path, "path", "p", "/", "the path to the jsonnet to reconcile")
	createFlags.StringVar(&sourceRef.Name, "source-name", "", "the name of the source object containing the jsonnet code")
	createFlags.StringVar(&sourceRef.Namespace, "source-namespace", "", "the namespace of the source object containing the jsonnet code (defaults to the creation namespace)")
	createFlags.StringVar(&sourceRef.Kind, "source-kind", "GitRepository", "the kind of source provided by --source-name.")
	createFlags.StringVarP(&createSpec.Namespace, "namespace", "n", "default", "The namespace to create the resource")
	createFlags.DurationVar(&createSpec.Spec.Interval.Duration, "interval", time.Minute*5, "the interval to reconcile the konfiguration")
	createFlags.DurationVar(&createSpec.Spec.RetryInterval.Duration, "retry-interval", time.Duration(0), "the interval to reconcile the konfiguration")
	createFlags.DurationVar(&createSpec.Spec.Timeout.Duration, "timeout", time.Duration(0), "the timeout for konfiguration reconcile attempts")
	createFlags.StringArrayVar(&createSpec.Spec.JsonnetPaths, "jsonnet-path", nil, "jsonnet paths to include in the invocation")
	createFlags.StringArrayVar(&createSpec.Spec.JsonnetURLs, "jsonnet-url", nil, "jsonnet urls to include in the invocation")
	createFlags.StringToStringVar(&createSpec.Spec.Variables.ExtStr, "ext-str", nil, "external variables declared as strings")
	createFlags.StringToStringVar(&createSpec.Spec.Variables.ExtCode, "ext-code", nil, "external variables declared as jsonnet code")
	createFlags.StringToStringVar(&createSpec.Spec.Variables.TLAStr, "tla-str", nil, "top-level arguments declared as strings")
	createFlags.StringToStringVar(&createSpec.Spec.Variables.TLACode, "tla-code", nil, "top-level arguments declared as jsonnet code")
	createFlags.StringVar(&injectFile, "inject", "", "a file containing jsonnet code to inject at the end of the evaluation")
	createFlags.StringVar(&konfigKubeConfig, "kubeconfig-secret", "", "a secret contaning a 'value' with a kubeconfig to use for this konfiguration")
	createFlags.StringVar(&createSpec.Spec.ServiceAccountName, "service-account", "", "a service account to impersonate for this konfiguration")
	createFlags.BoolVar(&createSpec.Spec.Prune, "prune", false, "whether to garbage collect orphaned resources")
	createFlags.BoolVar(&createSpec.Spec.Suspend, "suspended", false, "whether to start the konfiguration suspended")
	createFlags.BoolVar(&createSpec.Spec.Validate, "validate", false, "whether to validate resources against the server schema before applying")

	createFlags.BoolVar(&createExport, "export", false, "don't create the konfiguration, dump it's contents to stdout (e.g. to pipe to build)")

	rootCmd.AddCommand(createCmd)
}

var createCmd = &cobra.Command{
	Use:     "create <NAME>",
	Short:   "Create a new Konfiguration",
	Aliases: []string{"new"},
	Args:    cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		createSpec.Name = args[0]

		if !strings.HasPrefix(createSpec.Spec.Path, "http") && sourceRef.Name == "" {
			return errors.New("you must specify a full HTTP URL for --path when not providing a --source-name")
		}
		if sourceRef.Name != "" {
			if sourceRef.Namespace == "" {
				sourceRef.Namespace = createSpec.Namespace
			}
			createSpec.Spec.SourceRef = &sourceRef
		}
		if createSpec.Spec.RetryInterval.Duration == 0 {
			createSpec.Spec.RetryInterval = nil
		}
		if createSpec.Spec.Timeout.Duration == 0 {
			createSpec.Spec.Timeout = nil
		}
		if konfigKubeConfig != "" {
			createSpec.Spec.KubeConfig = &konfigurationv1.KubeConfig{
				SecretRef: corev1.LocalObjectReference{
					Name: konfigKubeConfig,
				},
			}
		}
		if injectFile != "" {
			inject, err := ioutil.ReadFile(injectFile)
			if err != nil {
				return err
			}
			createSpec.Spec.Inject = string(inject)
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if createExport {
			return serializer.Encode(&createSpec, os.Stdout)
		}
		if err := k8sClient.Create(context.Background(), &createSpec); err != nil {
			return err
		}
		fmt.Printf("Konfiguration %s/%s created\n", createSpec.GetName(), createSpec.GetNamespace())
		return nil
	},
}
