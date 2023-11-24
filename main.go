package main

import (
	"os"

	"github.com/aclevername/kratix-backstage-generator-pipeline/lib"
)

func main() {
	kratixDir := os.Getenv("KRATIX_DIR")
	if kratixDir == "" {
		kratixDir = "/kratix"
	}

	err := lib.Generate(kratixDir, os.Getenv("KRATIX_WORKFLOW_TYPE"), os.Getenv("KRATIX_PROMISE"))
	if err != nil {
		panic(err)
	}
}
