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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"path/filepath"
	"strings"

	securejoin "github.com/cyphar/filepath-securejoin"
	jsonnet "github.com/google/go-jsonnet"

	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/log"

	konfigurationv1 "github.com/pelotech/jsonnet-controller/api/v1"
)

// Builder is the main interface for rendering jsonnet to Kubernetes manifests.
type Builder interface {
	// Build will render the jsonnet at the given path using the configurations in the supplied konfig.
	// The restMapper is used to determine if objects are registered with the target API and their
	// cluster scope. A nil restMapper can be provided to skip this process.
	Build(ctx context.Context, restMapper meta.RESTMapper, path string) (*BuildOutput, error)
}

// NewBuilder constructs a jsonnet builder according to the konfiguration.
// Assets fetched over HTTP will be cached to the cacheDir.
func NewBuilder(konfig *konfigurationv1.Konfiguration, workdir, cacheDir string) (Builder, error) {
	b := &builder{vm: jsonnet.MakeVM(), cacheDir: cacheDir, konfig: konfig}

	// Register native functions
	registerNativeFuncs(b.vm)

	// Special URL scheme for embedded content
	searchURLs := []*url.URL{
		{Scheme: "internal", Path: "/"},
	}

	// Configure jsonnet paths in the workdir
	if paths := konfig.GetJsonnetPaths(); len(paths) > 0 {
		for _, path := range paths {
			joined, err := securejoin.SecureJoin(workdir, path)
			if err != nil {
				return nil, err
			}
			abs, err := filepath.Abs(joined)
			if err != nil {
				return nil, err
			}
			path = filepath.ToSlash(abs)
			if path[len(path)-1] != '/' {
				// trailing slash is important
				path = path + "/"
			}
			searchURLs = append(searchURLs, &url.URL{Scheme: "file", Path: path})
		}
	}

	// Configure remote jsonnet paths
	if urls := konfig.GetJsonnetURLs(); len(urls) > 0 {
		for _, ustr := range urls {
			u, err := url.Parse(ustr)
			if err != nil {
				return nil, err
			}
			if u.Path[len(u.Path)-1] != '/' {
				u.Path = u.Path + "/"
			}
			searchURLs = append(searchURLs, u)
		}
	}

	b.searchURLs = searchURLs

	// Inject any variables into the VM
	if vars := konfig.GetVariables(); vars != nil {
		vars.InjectInto(b.vm)
	}

	return b, nil
}

// builder implements the builder interface.
type builder struct {
	konfig     *konfigurationv1.Konfiguration
	searchURLs []*url.URL
	cacheDir   string
	vm         *jsonnet.VM
}

func (b *builder) Build(ctx context.Context, restMapper meta.RESTMapper, path string) (*BuildOutput, error) {
	// Configure the importer
	log := log.FromContext(ctx)
	b.vm.Importer(MakeUniversalImporter(log, b.searchURLs, b.cacheDir))

	// Evaluate the jsonnet
	evaluated, err := b.evaluateJsonnet(path)
	if err != nil {
		return nil, err
	}

	// Unmarshal the output
	var root interface{}
	if err := json.Unmarshal([]byte(evaluated), &root); err != nil {
		return nil, err
	}

	// Walk the output for kubernetes objects
	objs, err := jsonWalk(&walkContext{label: "<top>"}, root)
	if err != nil {
		return nil, err
	}

	output := newBuildOutput()

	// Build the output, taking care to ensure namespaces are properly set on
	// namespaced objects.
	for _, v := range objs {
		obj := &unstructured.Unstructured{Object: v.(map[string]interface{})}
		if obj.IsList() {
			list, err := obj.ToList()
			if err != nil {
				return nil, err
			}
			for _, o := range list.Items {
				if err := b.checkNamespace(restMapper, &o); err != nil {
					return nil, err
				}
				output.append(&o)
			}
			continue
		}
		if err := b.checkNamespace(restMapper, obj); err != nil {
			return nil, err
		}
		output.append(obj)
	}

	return output, nil
}

func (b *builder) evaluateJsonnet(path string) (string, error) {
	u, err := url.Parse(path)
	if err != nil {
		return "", err
	}
	if u.Scheme == "" {
		u.Scheme = "file"
	}
	if u.Scheme == "file" {
		abs, err := filepath.Abs(u.Path)
		if err != nil {
			return "", err
		}
		u.Path = abs
	}
	path = u.String()
	ext := filepath.Ext(path)
	// Double any single quotes in the path so they are quoted in the produced expression
	quotedPath := strings.Replace(path, "'", "''", -1)
	var expr string
	switch ext {
	case ".json":
		expr = fmt.Sprintf(`(import "internal:///kubecfg.libsonnet").parseJson(importstr @'%s')`, quotedPath)
	case ".yaml":
		expr = fmt.Sprintf(`(import "internal:///kubecfg.libsonnet").parseYaml(importstr @'%s')`, quotedPath)
	case ".jsonnet", ".libsonnet":
		expr = fmt.Sprintf("(import @'%s')", quotedPath)
	default:
		// Assume jsonnet - we are, after all, a jsonnet-controller
		expr = fmt.Sprintf("(import @'%s')", quotedPath)
	}

	// Add any user-defined injections
	expr += b.konfig.GetInjectSnippet()

	output, err := b.vm.EvaluateAnonymousSnippet("", expr)
	if err != nil {
		return "", errors.New(strings.TrimSpace(err.Error()))
	}
	return output, nil
}

func (b *builder) checkNamespace(restMapper meta.RESTMapper, obj *unstructured.Unstructured) error {
	if restMapper == nil {
		return nil
	}
	// retrieve the rest mapping for this gvk
	gvk := obj.GroupVersionKind()
	restMapping, err := restMapper.RESTMapping(schema.GroupKind{
		Group: gvk.Group,
		Kind:  gvk.Kind,
	}, gvk.Version)
	if err != nil {
		return err
	}
	// if it is a namespaced object, make sure there is a namespace defined
	if restMapping.Scope.Name() == meta.RESTScopeNameNamespace {
		if obj.GetNamespace() == "" {
			obj.SetNamespace(b.konfig.GetNamespace())
		}
	}
	return nil
}
