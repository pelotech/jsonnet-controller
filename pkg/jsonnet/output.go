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
	"sort"

	goyaml "gopkg.in/yaml.v2"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// BuildOutput contains the output from a build operation.
type BuildOutput struct {
	objects ObjectSorter

	// whether we sorted already
	sorted bool
	// cached yaml stream
	yamlStream []byte
}

// newBuildOutput creates a new empty build output.
func newBuildOutput() *BuildOutput {
	return &BuildOutput{objects: make(ObjectSorter, 0)}
}

// append adds an object to the build output.
func (b *BuildOutput) append(obj *unstructured.Unstructured) {
	b.objects = append(b.objects, obj)
}

// YAMLStream produces a yaml stream of the objects in this build output. The stream
// is cached internally so modifications to this output will not affect the produced
// stream from the first call.
func (b *BuildOutput) YAMLStream() ([]byte, error) {
	if b.yamlStream != nil {
		return b.yamlStream, nil
	}
	stream, err := toYAMLStream(b.SortedObjects())
	if err != nil {
		return nil, err
	}
	b.yamlStream = stream
	return b.yamlStream, nil
}

// SortedObjects returns a sorted list of objects in this output. Objects are sorted
// in the following order:
// - Namespaces (alphabetically)
// - CustomResourceDefinitions (alphabetically)
// - Resource namespaced names (alphabetically)
func (b *BuildOutput) SortedObjects() []*unstructured.Unstructured {
	if b.sorted {
		return b.objects
	}
	sort.Sort(b.objects)
	b.sorted = true
	return b.objects
}

func toYAMLStream(objects ObjectSorter) ([]byte, error) {
	stream := "---\n"

	for i, obj := range objects {
		out, err := goyaml.Marshal(obj.Object)
		if err != nil {
			return nil, err
		}
		stream += string(out)
		if i == len(objects)-1 {
			break
		}
		stream += "\n---\n"
	}

	return []byte(stream), nil
}
