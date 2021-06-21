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
	_ "embed"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/spf13/cobra"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
)

//go:embed manifest.yaml
var manifest string

// Populated by CI during a release build
var Version string

var exportInstall bool
var installNamespace string
var installVersion string

func init() {
	installCmd.Flags().BoolVar(&exportInstall, "export", false, "dump installation manifests without installing to the cluster (ignores --namespace)")
	installCmd.Flags().StringVar(&installNamespace, "namespace", "flux-system", "the namespace to install the jsonnet-controller to")
	installCmd.Flags().StringVar(&installVersion, "version", Version, "the version of the controller to install")

	rootCmd.AddCommand(installCmd)
}

var imageRegex = regexp.MustCompile("image: (ghcr.io/pelotech/jsonnet-controller):(latest)")

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the jsonnet controller into a cluster",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if !exportInstall {
			if err := checkClient(); err != nil {
				return err
			}
		}
		if installVersion != "" {
			manifest = imageRegex.ReplaceAllString(manifest, fmt.Sprintf("image: $1:%s", installVersion))
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if exportInstall {
			fmt.Println(string(manifest))
			return nil
		}
		reader := yaml.NewYAMLOrJSONDecoder(strings.NewReader(manifest), 2048)
		for {
			toCreate := &unstructured.Unstructured{}
			err := reader.Decode(&toCreate)
			if err != nil {
				if err == io.EOF {
					return nil
				}
				return err
			}
			// If using a custom install namespace, set the namespace
			gvk := toCreate.GroupVersionKind()
			if installNamespace != "flux-system" {
				restMapping, err := k8sClient.RESTMapper().RESTMapping(schema.GroupKind{
					Group: gvk.Group,
					Kind:  gvk.Kind,
				}, gvk.Version)
				if err != nil {
					return err
				}
				if restMapping.Scope.Name() == meta.RESTScopeNameNamespace {
					toCreate.SetNamespace(installNamespace)
				}
			}
			id := fmt.Sprintf("%s '%s/%s'", gvk.Kind, toCreate.GetNamespace(), toCreate.GetName())
			fmt.Println("Creating", id)
			if err := k8sClient.Create(cmd.Context(), toCreate); err != nil {
				return err
			}
		}
	},
}
