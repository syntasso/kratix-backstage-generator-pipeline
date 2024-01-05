package lib

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/syntasso/kratix/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type Opts struct {
	RunningInPipeline bool
	OutputDirectory   string
	MetadataDirectory string
	Input             string
	//if its not a Promise its assumed its a resource request
	IsPromise   bool
	PromiseName string
}

func Generate(o Opts) error {
	f, err := os.Open(o.Input)
	if err != nil {
		return fmt.Errorf("failed to open input file %s: %w", o.Input, err)
	}
	defer f.Close()

	decoder := yaml.NewYAMLOrJSONDecoder(f, 2048)

	if o.RunningInPipeline {
		err = os.WriteFile(filepath.Join(o.MetadataDirectory, "destination-selectors.yaml"), []byte(`- directory: backstage
  matchLabels:
    environment: backstage`), 0777)
		if err != nil {
			return fmt.Errorf("failed to write scheduling file: %w", err)
		}
	}

	if o.IsPromise {
		promise := &v1alpha1.Promise{}
		err = decoder.Decode(promise)
		if err != nil {
			return err
		}
		err = generatePromiseEntities(o, promise)

		if err != nil {
			return err
		}
		return nil
	}

	object := &unstructured.Unstructured{}
	err = decoder.Decode(object)
	if err != nil {
		return err
	}

	err = generateResourceEntities(o, object)
	if err != nil {
		return err
	}
	return nil
}

func generatePromiseEntities(o Opts, promise *v1alpha1.Promise) error {
	componentBytes := []byte(fmt.Sprintf(`---
apiVersion: backstage.io/v1alpha1
kind: Component
metadata:
  annotations:
    backstage.io/kubernetes-id: %[1]s
  description: Create a %[1]s
  links:
  - icon: help
    title: Support
    url: https://github.com/syntasso/kratix-backstage
  name: %[1]s
  title: %[1]s Promise
spec:
  dependsOn:
  - component:default/kratix
  lifecycle: production
  owner: kratix-platform
  providesApis:
  - %[1]s-promise-api
  type: promise
`, promise.GetName())) //TODO change last arg to be upper case

	backstageDir := filepath.Join(o.OutputDirectory, "backstage")
	err := os.MkdirAll(backstageDir, 0777)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(backstageDir, promise.GetName()+"-component.yaml"), componentBytes, 0777)
	if err != nil {
		return fmt.Errorf("failed to write component file: %w", err)
	}

	return generateTemplate(o, promise)
}

func generateResourceEntities(o Opts, us *unstructured.Unstructured) error {
	componentBytes := []byte(fmt.Sprintf(`---
apiVersion: backstage.io/v1alpha1
kind: Component
metadata:
  name: %[1]s-%[2]s
  title: "%[1]s %[2]s"
  description: %[2]s created via %[1]s Promise
  annotations:
    backstage.io/kubernetes-label-selector: %[1]s-cr=%[2]s
  links:
  - url: https://github.com/syntasso/kratix-backstage
    title: Support
    icon: help
spec:
  type: service
  lifecycle: production
  owner: kratix-worker
  dependsOn:
    - component:default/%[1]s
  providesApis:
    - namespace-server-api
`, o.PromiseName, us.GetName())) //TODO change last arg to be upper case

	backstageDir := filepath.Join(o.OutputDirectory, "backstage")
	err := os.MkdirAll(backstageDir, 0777)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(backstageDir, us.GetName()+"-component.yaml"), componentBytes, 0777)
	if err != nil {
		return fmt.Errorf("failed to write component file: %w", err)
	}

	return nil
}
