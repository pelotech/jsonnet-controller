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
	"fmt"
	"os"

	"github.com/spf13/cobra"

	konfigurationv1 "github.com/pelotech/jsonnet-controller/api/v1beta1"
	"github.com/pelotech/jsonnet-controller/pkg/jsonnet"
)

var showKonfig = &konfigurationv1.Konfiguration{
	Spec: konfigurationv1.KonfigurationSpec{
		// TODO: there are more options that should be exposed
		Variables: &konfigurationv1.Variables{
			ExtStr:  map[string]string{},
			ExtCode: map[string]string{},
			TLAStr:  map[string]string{},
			TLACode: map[string]string{},
		},
	},
}

func init() {
	flags := showCommand.Flags()

	flags.StringToStringVar(&showKonfig.Spec.Variables.ExtStr, "ext-str", nil, "external string variables")
	flags.StringToStringVar(&showKonfig.Spec.Variables.ExtCode, "ext-code", nil, "external code variables")
	flags.StringToStringVar(&showKonfig.Spec.Variables.TLAStr, "tla-str", nil, "top-level string variables")
	flags.StringToStringVar(&showKonfig.Spec.Variables.TLACode, "tla-code", nil, "top-level code variables")

	rootCmd.AddCommand(showCommand)
}

var showCommand = &cobra.Command{
	Use:   "show [FILE]",
	Short: "Do a client-side evaluation of a jsonnet file or path",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cwd, err := os.Getwd()
		if err != nil {
			return err
		}
		builder, err := jsonnet.NewBuilder(showKonfig, cwd, "")
		if err != nil {
			return err
		}
		out, err := builder.Build(context.Background(), nil, args[0])
		if err != nil {
			return err
		}
		stream, err := out.YAMLStream()
		if err != nil {
			return err
		}
		fmt.Println(string(stream))
		return nil
	},
}
