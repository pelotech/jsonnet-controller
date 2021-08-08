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

package jsonnet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	jsonnet "github.com/google/go-jsonnet"
	"github.com/google/go-jsonnet/ast"
	"gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/engine"
)

// DefaultNameFormat to use when no nameFormat is supplied
const DefaultNameFormat = `{{ print .kind "_" .metadata.name | snakecase }}`

type HelmTemplateOpts struct {
	// Values to pass to Helm using --set
	Values map[string]interface{} `json:"values"`
	// Values files to pass to helm using --values
	ValuesFiles []string `json:"valuesFiles"`
	// Namespace scope for this request
	Namespace string `json:"namespace"`

	// CalledFrom is the file that calls helmTemplate. This is used to find the
	// vendored chart relative to this file
	CalledFrom string `json:"calledFrom"`
	// NameTemplate is used to create the keys in the resulting map
	NameFormat string `json:"nameFormat"`
}

func helmTemplateNativeFunc() *jsonnet.NativeFunction {
	return &jsonnet.NativeFunction{
		Name:   "helmTemplate",
		Params: []ast.Identifier{"name", "chart", "opts"},
		Func: func(data []interface{}) (interface{}, error) {
			name, ok := data[0].(string)
			if !ok {
				return nil, fmt.Errorf("first argument 'name' must be of 'string' type, got '%T' instead", data[0])
			}

			chartPath, ok := data[1].(string)
			if !ok {
				return nil, fmt.Errorf("second argument 'chart' must be of 'string' type, got '%T' instead", data[1])
			}

			opts, err := parseHelmTemplatOpts(data[2])
			if err != nil {
				return nil, err
			}

			chart, err := loader.Load(chartPath)
			if err != nil {
				return nil, err
			}

			helmVals := make(chartutil.Values)
			for _, f := range opts.ValuesFiles {
				vals, err := chartutil.ReadValuesFile(f)
				if err != nil {
					return nil, err
				}
				for k, v := range vals {
					helmVals[k] = v
				}
			}
			if len(opts.Values) > 0 {
				valueData, err := yaml.Marshal(opts.Values)
				if err != nil {
					return nil, err
				}
				vals, err := chartutil.ReadValues(valueData)
				if err != nil {
					return nil, err
				}
				for k, v := range vals {
					helmVals[k] = v
				}
			}

			if err := chartutil.ProcessDependencies(chart, helmVals); err != nil {
				return nil, err
			}

			options := chartutil.ReleaseOptions{
				Name:      name,
				Namespace: opts.Namespace,
				Revision:  1,
				IsInstall: true,
				IsUpgrade: false,
			}
			valuesToRender, err := chartutil.ToRenderValues(chart, helmVals, options, nil)
			if err != nil {
				return nil, err
			}

			objects, err := engine.Render(chart, valuesToRender)
			if err != nil {
				return nil, err
			}

			return helmObjectsToOutput(opts, objects)
		},
	}
}

func helmObjectsToOutput(opts *HelmTemplateOpts, rendered map[string]string) (string, error) {
	out := make(map[string]interface{})
	for fname, content := range rendered {
		// Skip notes files
		if strings.HasSuffix(fname, "NOTES.txt") {
			continue
		}
		if strings.TrimSpace(content) == "" {
			continue
		}
		var obj map[string]interface{}
		err := yaml.Unmarshal([]byte(content), &obj)
		if err != nil {
			return "", err
		}
		if opts.NameFormat == "" {
			opts.NameFormat = DefaultNameFormat
		}
		var buf bytes.Buffer
		tmpl, err := template.New("").Funcs(sprig.HermeticTxtFuncMap()).Parse(opts.NameFormat)
		if err != nil {
			return "", err
		}
		if err := tmpl.Execute(&buf, obj); err != nil {
			return "", err
		}
		out[buf.String()] = obj
	}
	body, err := json.Marshal(out)
	return string(body), err
}

func parseHelmTemplatOpts(data interface{}) (*HelmTemplateOpts, error) {
	c, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	var opts HelmTemplateOpts
	if err := json.Unmarshal(c, &opts); err != nil {
		return nil, err
	}

	// Charts are only allowed at relative paths. Use conf.CalledFrom to find the callers directory
	// if opts.CalledFrom == "" {
	// 	return nil, fmt.Errorf("helmTemplate: 'opts.calledFrom' is unset or empty.\nTanka needs this to find your charts. See https://tanka.dev/helm#optscalledfrom-unset\n")
	// }

	return &opts, nil
}
