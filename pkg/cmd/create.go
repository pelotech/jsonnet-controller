package cmd

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
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

var konfigKubeConfig string
var sourceName, sourceNamespace string
var createExport bool

func init() {
	flags := createCmd.Flags()

	flags.StringVarP(&createSpec.Spec.Path, "path", "p", "/", "the path to the jsonnet to reconcile")
	flags.StringVar(&sourceName, "source-name", "", "the name of the GitRepository containing the jsonnet code")
	flags.StringVar(&sourceNamespace, "source-namespace", "", "the namespace of the GitRepository containing the jsonnet code (defaults to the creation namespace)")

	flags.StringVarP(&createSpec.Namespace, "namespace", "n", "default", "The namespace to create the resource")
	flags.DurationVar(&createSpec.Spec.Interval.Duration, "interval", time.Minute*5, "the interval to reconcile the konfiguration")
	flags.DurationVar(&createSpec.Spec.RetryInterval.Duration, "retry-interval", time.Duration(0), "the interval to reconcile the konfiguration")
	flags.DurationVar(&createSpec.Spec.Timeout.Duration, "timeout", time.Duration(0), "the timeout for konfiguration reconcile attempts")
	flags.StringArrayVar(&createSpec.Spec.JsonnetPaths, "jsonnet-path", nil, "jsonnet paths to include in the invocation")
	flags.StringArrayVar(&createSpec.Spec.JsonnetURLs, "jsonnet-url", nil, "jsonnet urls to include in the invocation")

	flags.StringToStringVar(&createSpec.Spec.Variables.ExtStr, "ext-str", nil, "external variables declared as strings")
	flags.StringToStringVar(&createSpec.Spec.Variables.ExtCode, "ext-code", nil, "external variables declared as jsonnet code")
	flags.StringToStringVar(&createSpec.Spec.Variables.TLAStr, "tla-str", nil, "top-level arguments declared as strings")
	flags.StringToStringVar(&createSpec.Spec.Variables.TLACode, "tla-code", nil, "top-level arguments declared as jsonnet code")
	flags.StringVar(&createSpec.Spec.Inject, "inject", "", "a file containing jsonnet code to inject at the end of the evaluation")

	flags.StringVar(&konfigKubeConfig, "kubeconfig-secret", "", "a secret contaning a 'value' with a kubeconfig to use for this konfiguration")
	flags.StringVar(&createSpec.Spec.ServiceAccountName, "service-account", "", "a service account to impersonate for this konfiguration")
	flags.BoolVar(&createSpec.Spec.Prune, "prune", false, "whether to garbage collect orphaned resources")
	flag.BoolVar(&createSpec.Spec.Suspend, "suspended", false, "whether to start the konfiguration suspended")
	flag.BoolVar(&createSpec.Spec.Validate, "validate", false, "whether to validate resources against the server schema before applying")

	flags.BoolVar(&createExport, "export", false, "don't create the konfiguration dump it's contents to stdout")

	rootCmd.AddCommand(createCmd)
}

var createCmd = &cobra.Command{
	Use:     "create <NAME>",
	Short:   "Create a new Konfiguration",
	Aliases: []string{"new"},
	Args:    cobra.ExactArgs(1),
	PreRunE: func(cmd *cobra.Command, args []string) error {
		createSpec.Name = args[0]

		if createSpec.Spec.Path == "/" && sourceName == "" {
			return errors.New("you must specify a full HTTP URL for --path when not providing a --source-name and/or --source-namespace")
		}
		if sourceName != "" {
			if sourceNamespace == "" {
				sourceNamespace = createSpec.Namespace
			}
			createSpec.Spec.SourceRef = &meta.NamespacedObjectKindReference{
				Kind:      "GitRepository",
				Name:      sourceName,
				Namespace: sourceNamespace,
			}
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
