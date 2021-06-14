/*
Copyright 2017 The kubecfg authors


   Licensed under the Apache License, Version 2.0 (the "License");
   you may not use this file except in compliance with the License.
   You may obtain a copy of the License at

     http://www.apache.org/licenses/LICENSE-2.0

   Unless required by applicable law or agreed to in writing, software
   distributed under the License is distributed on an "AS IS" BASIS,
   WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
   See the License for the specific language governing permissions and
   limitations under the License.

Copyright 2021 Pelotech
	- Removed resolveImage function
*/

package jsonnet

import (
	"bytes"
	"encoding/json"
	"io"
	"regexp"
	"strings"

	jsonnet "github.com/google/go-jsonnet"
	jsonnetAst "github.com/google/go-jsonnet/ast"
	goyaml "gopkg.in/yaml.v2"

	"k8s.io/apimachinery/pkg/util/yaml"
)

// registerNativeFuncs adds kubecfg's native jsonnet functions to provided VM
func registerNativeFuncs(vm *jsonnet.VM) {
	// TODO(mkm): go-jsonnet 0.12.x now contains native std.parseJson; deprecate and remove this one.
	vm.NativeFunction(&jsonnet.NativeFunction{
		Name:   "parseJson",
		Params: []jsonnetAst.Identifier{"json"},
		Func: func(args []interface{}) (res interface{}, err error) {
			data := []byte(args[0].(string))
			err = json.Unmarshal(data, &res)
			return
		},
	})

	vm.NativeFunction(&jsonnet.NativeFunction{
		Name:   "parseYaml",
		Params: []jsonnetAst.Identifier{"yaml"},
		Func: func(args []interface{}) (res interface{}, err error) {
			ret := []interface{}{}
			data := []byte(args[0].(string))
			d := yaml.NewYAMLToJSONDecoder(bytes.NewReader(data))
			for {
				var doc interface{}
				if err := d.Decode(&doc); err != nil {
					if err == io.EOF {
						break
					}
					return nil, err
				}
				ret = append(ret, doc)
			}
			return ret, nil
		},
	})

	vm.NativeFunction(&jsonnet.NativeFunction{
		Name:   "manifestJson",
		Params: []jsonnetAst.Identifier{"json", "indent"},
		Func: func(args []interface{}) (res interface{}, err error) {
			value := args[0]
			indent := int(args[1].(float64))
			data, err := json.MarshalIndent(value, "", strings.Repeat(" ", indent))
			if err != nil {
				return "", err
			}
			data = append(data, byte('\n'))
			return string(data), nil
		},
	})

	vm.NativeFunction(&jsonnet.NativeFunction{
		Name:   "manifestYaml",
		Params: []jsonnetAst.Identifier{"json"},
		Func: func(args []interface{}) (res interface{}, err error) {
			value := args[0]
			output, err := goyaml.Marshal(value)
			return string(output), err
		},
	})

	vm.NativeFunction(&jsonnet.NativeFunction{
		Name:   "escapeStringRegex",
		Params: []jsonnetAst.Identifier{"str"},
		Func: func(args []interface{}) (res interface{}, err error) {
			return regexp.QuoteMeta(args[0].(string)), nil
		},
	})

	vm.NativeFunction(&jsonnet.NativeFunction{
		Name:   "regexMatch",
		Params: []jsonnetAst.Identifier{"regex", "string"},
		Func: func(args []interface{}) (res interface{}, err error) {
			return regexp.MatchString(args[0].(string), args[1].(string))
		},
	})

	vm.NativeFunction(&jsonnet.NativeFunction{
		Name:   "regexSubst",
		Params: []jsonnetAst.Identifier{"regex", "src", "repl"},
		Func: func(args []interface{}) (res interface{}, err error) {
			regex := args[0].(string)
			src := args[1].(string)
			repl := args[2].(string)

			r, err := regexp.Compile(regex)
			if err != nil {
				return "", err
			}
			return r.ReplaceAllString(src, repl), nil
		},
	})
}
