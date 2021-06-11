package controllers

import (
	"bytes"
	"context"
	"crypto/sha1"
	"fmt"
	"io"
	"sort"

	"github.com/go-logr/logr"
	goyaml "gopkg.in/yaml.v2"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"

	appsv1 "github.com/pelotech/kubecfg-operator/api/v1"
)

type ObjectSorter []unstructured.Unstructured

func (o ObjectSorter) Len() int      { return len(o) }
func (o ObjectSorter) Swap(i, j int) { o[i], o[j] = o[j], o[i] }
func (o ObjectSorter) Less(i, j int) bool {
	return fmt.Sprintf("%s/%s", o[i].GetNamespace(), o[i].GetName()) <
		fmt.Sprintf("%s/%s", o[j].GetNamespace(), o[j].GetName())
}

func (k *KonfigurationReconciler) computeChecksum(ctx context.Context, log logr.Logger, konfig *appsv1.Konfiguration, path string) ([]byte, string, error) {
	showOutput, err := runKubecfgShow(ctx, log, konfig, path)
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
			objects = append(objects, objList.Items...)
		} else {
			objects = append(objects, obj)
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
		sortedStream += "\n---"
	}

	h := sha1.New()
	io.WriteString(h, sortedStream)

	return []byte(sortedStream), fmt.Sprintf("%x", h.Sum(nil)), nil
}
