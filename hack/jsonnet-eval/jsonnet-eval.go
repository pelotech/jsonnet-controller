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
