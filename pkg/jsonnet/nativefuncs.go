package jsonnet

import (
	"bytes"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"regexp"
	"strings"

	"github.com/Masterminds/semver"
	jsonnet "github.com/google/go-jsonnet"
	jsonnetAst "github.com/google/go-jsonnet/ast"
	goyaml "gopkg.in/yaml.v2"

	"k8s.io/apimachinery/pkg/util/yaml"
)

// registerNativeFuncs adds kubecfg's native jsonnet functions to provided VM
func registerNativeFuncs(vm *jsonnet.VM) {

	// Version Compare Functions
	// https://masterminds.github.io/sprig/semver.html

	vm.NativeFunction(&jsonnet.NativeFunction{
		Name:   "semver",
		Params: []jsonnetAst.Identifier{"version"},
		Func: func(args []interface{}) (res interface{}, err error) {
			in := args[0].(string)
			vers, err := semver.NewVersion(in)
			if err != nil {
				return
			}
			res = map[string]interface{}{
				"major":       float64(vers.Major()),
				"minor":       float64(vers.Minor()),
				"patch":       float64(vers.Patch()),
				"pre_release": vers.Prerelease(),
				"metadata":    vers.Metadata(),
			}
			return
		},
	})

	vm.NativeFunction(&jsonnet.NativeFunction{
		Name:   "semverCompare",
		Params: []jsonnetAst.Identifier{"constraint", "version"},
		Func: func(args []interface{}) (res interface{}, err error) {
			constraint := args[0].(string)
			version := args[1].(string)
			vers, err := semver.NewVersion(version)
			if err != nil {
				return
			}
			c, err := semver.NewConstraint(constraint)
			if err != nil {
				return
			}
			return c.Check(vers), nil
		},
	})

	// Hashing Functions

	vm.NativeFunction(&jsonnet.NativeFunction{
		Name:   "sha1Sum",
		Params: []jsonnetAst.Identifier{"str"},
		Func: func(args []interface{}) (res interface{}, err error) {
			in := args[0].(string)
			h := sha1.New()
			_, err = h.Write([]byte(in))
			if err != nil {
				return
			}
			res = fmt.Sprintf("%x", h.Sum(nil))
			return
		},
	})

	vm.NativeFunction(&jsonnet.NativeFunction{
		Name:   "sha256Sum",
		Params: []jsonnetAst.Identifier{"str"},
		Func: func(args []interface{}) (res interface{}, err error) {
			in := args[0].(string)
			h := sha256.New()
			_, err = h.Write([]byte(in))
			if err != nil {
				return
			}
			res = fmt.Sprintf("%x", h.Sum(nil))
			return
		},
	})

	// JSON/YAML Parsing

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

	// Regex Functions

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
