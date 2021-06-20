package cmd

import (
	"bytes"
	_ "embed"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/yaml"
)

//go:embed manifest.yaml
var manifest []byte

var exportInstall bool
var installNamespace string

func init() {
	installCmd.Flags().BoolVar(&exportInstall, "export", false, "dump installation manifests without installing to the cluster (ignores --namespace)")
	installCmd.Flags().StringVar(&installNamespace, "namespace", "flux-system", "the namespace to install the jsonnet-controller to")

	rootCmd.AddCommand(installCmd)
}

var installCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the jsonnet controller into a cluster",
	PreRunE: func(cmd *cobra.Command, args []string) error {
		if !exportInstall {
			return checkClient()
		}
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		if exportInstall {
			fmt.Println(string(manifest))
			return nil
		}
		reader := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(manifest), 2048)
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
