package main

import (
	"flag"
	"os"
	"path/filepath"

	"github.com/aclevername/kratix-backstage-generator-pipeline/lib"
)

func main() {
	runningInPipeline := flag.Bool("in-pipeline", false, "Whether the generator is running in a pipeline or not. If true no other flags are required.")
	outputDirectory := flag.String("output-directory", "", "When running out of pipeline, the directory to output the generated files to")
	input := flag.String("filepath", "", "Kubernetes Resource to generate Backstage files from. Only used when --in-pipeline=false")
	inputType := flag.String("file-type", "promise", "The type of input. Either 'promise' or 'resource'. Only used when --in-pipeline=false")
	promiseName := flag.String("promise-name", "", "The name of the promise. Only required when --file-type=resource")
	flag.Parse()

	opts := lib.Opts{}
	if *runningInPipeline {
		kratixDir := os.Getenv("KRATIX_DIR")
		if kratixDir == "" {
			kratixDir = "/kratix"
		}
		// os.Getenv("KRATIX_WORKFLOW_TYPE"), os.Getenv("KRATIX_PROMISE_NAME")
		opts = lib.Opts{
			RunningInPipeline: *runningInPipeline,
			Input:             filepath.Join(kratixDir, "input", "object.yaml"),
			MetadataDirectory: filepath.Join(kratixDir, "metadata"),
			IsPromise:         *inputType == "promise",
			OutputDirectory:   filepath.Join(kratixDir, "output"),
			PromiseName:       os.Getenv("KRATIX_PROMISE_NAME"),
		}
	} else {
		opts = lib.Opts{
			RunningInPipeline: *runningInPipeline,
			Input:             *input,
			OutputDirectory:   *outputDirectory,
			PromiseName:       *promiseName,
			IsPromise:         *inputType == "promise",
		}

	}
	err := lib.Generate(opts)
	if err != nil {
		panic(err)
	}
}
