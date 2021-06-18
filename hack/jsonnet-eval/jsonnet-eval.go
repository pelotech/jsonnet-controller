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

package main

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelotech/jsonnet-controller/pkg/jsonnet"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Println("USAGE: jsonnet-eval <path>")
		os.Exit(1)
	}

	cacheDir, err := os.UserCacheDir()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	builder, err := jsonnet.NewBuilder(nil, "", filepath.Join(cacheDir, "jsonnet"))
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	out, err := builder.Evaluate(os.Args[1])
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	fmt.Print(out)
}
