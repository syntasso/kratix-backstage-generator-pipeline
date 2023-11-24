package lib

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/syntasso/kratix/api/v1alpha1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/util/yaml"
)

func Generate(kratixDir, workflowType, promiseName string) error {
	kratixInputObject := filepath.Join(kratixDir, "input", "object.yaml")
	fmt.Println("KRATIX_INPUT_DIR: " + kratixInputObject)

	f, err := os.Open(kratixInputObject)
	if err != nil {
		return err
	}
	defer f.Close()

	decoder := yaml.NewYAMLOrJSONDecoder(f, 2048)

	err = os.WriteFile(filepath.Join(kratixDir, "metadata", "destination-selectors.yaml"), []byte(`- directory: backstage
  matchLabels:
    environment: backstage`), 0777)
	if err != nil {
		return fmt.Errorf("failed to write scheduling file: %w", err)
	}

	if workflowType == "promise" {
		promise := &v1alpha1.Promise{}
		err = decoder.Decode(promise)
		if err != nil {
			return err
		}
		err = generatePromiseEntities(kratixDir, promise)

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

	err = generateResourceEntities(kratixDir, promiseName, object)
	if err != nil {
		return err
	}
	return nil
}

func generatePromiseEntities(kratixDir string, promise *v1alpha1.Promise) error {
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

	backstageDir := filepath.Join(kratixDir, "output", "backstage")
	err := os.MkdirAll(backstageDir, 0777)
	if err != nil {
		return err
	}
	err = os.WriteFile(filepath.Join(backstageDir, promise.GetName()+"-component.yaml"), componentBytes, 0777)
	if err != nil {
		return fmt.Errorf("failed to write component file: %w", err)
	}

	return generateTemplate(kratixDir, promise)
}

func generateResourceEntities(kratixDir, promiseName string, us *unstructured.Unstructured) error {
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
`, promiseName, us.GetName())) //TODO change last arg to be upper case

	backstageDir := filepath.Join(kratixDir, "output", "backstage")
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
