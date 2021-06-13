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

package controllers

import (
	"bytes"
	"context"
	"crypto/sha1"
	"fmt"
	"io"
	"sort"

	goyaml "gopkg.in/yaml.v2"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"

	appsv1 "github.com/pelotech/kubecfg-operator/api/v1"
)

type ObjectSorter []*unstructured.Unstructured

func (o ObjectSorter) Len() int      { return len(o) }
func (o ObjectSorter) Swap(i, j int) { o[i], o[j] = o[j], o[i] }
func (o ObjectSorter) Less(i, j int) bool {
	return fmt.Sprintf("%s/%s", o[i].GetNamespace(), o[i].GetName()) <
		fmt.Sprintf("%s/%s", o[j].GetNamespace(), o[j].GetName())
}

func (k *KonfigurationReconciler) build(ctx context.Context, konfig *appsv1.Konfiguration, path string) ([]byte, string, error) {
	showOutput, err := runKubecfgShow(ctx, konfig, path)
	if err != nil {
		return nil, "", err
	}

	reader := yaml.NewYAMLOrJSONDecoder(bytes.NewReader(showOutput), 2048)
	objects := make(ObjectSorter, 0)
	for {
		var obj unstructured.Unstructured
		err := reader.Decode(&obj)
		if err == io.EOF {
			break
		} else if err != nil {
			return nil, "", err
		}
		if obj.IsList() {
			objList, err := obj.ToList()
			if err != nil {
				return nil, "", err
			}
			for _, o := range objList.Items {
				if o.GetNamespace() == "" {
					o.SetNamespace(konfig.GetNamespace())
				}
				objects = append(objects, &o)
			}
		} else {
			if obj.GetNamespace() == "" {
				obj.SetNamespace(konfig.GetNamespace())
			}
			objects = append(objects, &obj)
		}
	}

	sort.Sort(objects)

	sortedStream := "---\n"

	for i, obj := range objects {
		out, err := goyaml.Marshal(obj.Object)
		if err != nil {
			return nil, "", err
		}
		sortedStream += string(out)
		if i == len(objects)-1 {
			break
		}
		sortedStream += "\n---\n"
	}

	h := sha1.New()
	if _, err := io.WriteString(h, sortedStream); err != nil {
		return nil, "", err
	}

	return []byte(sortedStream), fmt.Sprintf("%x", h.Sum(nil)), nil
}