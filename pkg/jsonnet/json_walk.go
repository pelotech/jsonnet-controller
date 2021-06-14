// Copyright 2017 The kubecfg authors
//
//
//    Licensed under the Apache License, Version 2.0 (the "License");
//    you may not use this file except in compliance with the License.
//    You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS,
//    WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//    See the License for the specific language governing permissions and
//    limitations under the License.

package jsonnet

import "fmt"

type walkContext struct {
	parent *walkContext
	label  string
}

func (c *walkContext) String() string {
	parent := ""
	if c.parent != nil {
		parent = c.parent.String()
	}
	return parent + c.label
}

func jsonWalk(parentCtx *walkContext, obj interface{}) ([]interface{}, error) {
	switch o := obj.(type) {
	case nil:
		return []interface{}{}, nil
	case map[string]interface{}:
		if o["kind"] != nil && o["apiVersion"] != nil {
			return []interface{}{o}, nil
		}
		ret := []interface{}{}
		for k, v := range o {
			ctx := walkContext{
				parent: parentCtx,
				label:  "." + k,
			}
			children, err := jsonWalk(&ctx, v)
			if err != nil {
				return nil, err
			}
			ret = append(ret, children...)
		}
		return ret, nil
	case []interface{}:
		ret := make([]interface{}, 0, len(o))
		for i, v := range o {
			ctx := walkContext{
				parent: parentCtx,
				label:  fmt.Sprintf("[%d]", i),
			}
			children, err := jsonWalk(&ctx, v)
			if err != nil {
				return nil, err
			}
			ret = append(ret, children...)
		}
		return ret, nil
	default:
		return nil, fmt.Errorf("looking for kubernetes object at %s, but instead found %T", parentCtx, o)
	}
}
