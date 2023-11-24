package lib

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	v1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	yamlsig "sigs.k8s.io/yaml"

	"github.com/syntasso/kratix/api/v1alpha1"
)

type Template struct {
	metav1.TypeMeta `json:",inline,omitempty"`
	Metadata        Metadata     `json:"metadata,omitempty"`
	Spec            TemplateSpec `json:"spec,omitempty"`
}

type Metadata struct {
	metav1.ObjectMeta `json:",inline"`
	Description       string   `json:"description,omitempty"`
	Tags              []string `json:"tags,omitempty"`
	Title             string   `json:"title,omitempty"`
}

type TemplateSpec struct {
	Lifecycle  string      `json:"lifecycle,omitempty"`
	Owner      string      `json:"owner,omitempty"`
	Parameters []Parameter `json:"parameters,omitempty"`
	Steps      []Step      `json:"steps,omitempty"`
	Type       string      `json:"type,omitempty"`
}

type Step struct {
	Action string `json:"action,omitempty"`
	ID     string `json:"id,omitempty"`
	Input  Input  `json:"input,omitempty"`
	Name   string `json:"name,omitempty"`
}

type Input struct {
	Manifest   string `json:"manifest,omitempty"`
	Namespaced bool   `json:"namespaced,omitempty"`
}

type Parameter struct {
	Properties map[string]Properties `json:"properties,omitempty"`
	Required   []string              `json:"required,omitempty"`
	Title      string                `json:"title,omitempty"`
}

type Properties struct {
	Description string `json:"description,omitempty"`
	Title       string `json:"title,omitempty"`
	Type        string `json:"type,omitempty"`
}

type RR struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              map[string]string `json:"spec,omitempty"`
}

func generateTemplate(kratixDir string, promise *v1alpha1.Promise) error {
	rrCRD := &v1.CustomResourceDefinition{}
	if err := json.Unmarshal(promise.Spec.API.Raw, rrCRD); err != nil {
		return fmt.Errorf("api is not a valid CRD: %w", err)
	}

	template, err := generateBackstageTemplateWithoutProperties(rrCRD)
	if err != nil {
		return err
	}

	//Generate the manifest the kubectl plugin will apply based on the paremeters
	rrManifestTemplate := RR{
		TypeMeta: metav1.TypeMeta{
			APIVersion: rrCRD.Spec.Group + "/" + rrCRD.Spec.Versions[0].Name,
			Kind:       rrCRD.Spec.Names.Kind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "${{ parameters.name }}",
			Namespace: "${{ parameters.namespace}}",
			Labels: map[string]string{
				"backstage.io/kubernetes-id": rrCRD.Spec.Names.Kind,
			},
		},
		Spec: map[string]string{},
	}

	//Generate the parameter properties based on the CRD
	props := map[string]Properties{}
	for key, prop := range rrCRD.Spec.Versions[0].Schema.OpenAPIV3Schema.Properties["spec"].Properties {
		props[key] = Properties{
			Description: prop.Description,
			Title:       "Spec." + strings.Title(key),
			Type:        prop.Type,
		}
		rrManifestTemplate.Spec[key] = fmt.Sprintf("${{ parameters.%s }}", key)
	}

	props["namespace"] = Properties{
		Description: "Namespace for the request in the platform cluster",
		Title:       "Metadata.Namespace",
		Type:        "string",
	}

	props["name"] = Properties{
		Description: "Name for the request in the platform cluster",
		Title:       "Metadata.Name",
		Type:        "string",
	}
	fmt.Println(props)

	sampleRRBytes, err := yamlsig.Marshal(rrManifestTemplate)
	if err != nil {
		return err

	}

	template.Spec.Steps[0].Input.Manifest = string(sampleRRBytes)
	template.Spec.Parameters = []Parameter{
		{
			Properties: props,
			Required:   append(rrCRD.Spec.Versions[0].Schema.OpenAPIV3Schema.Properties["spec"].Required, "namespace", "name"),
			Title:      strings.Title(rrCRD.Spec.Names.Kind) + " as a Service",
		},
	}

	//Convert to bytes
	templateBytes, err := yamlsig.Marshal(template)
	if err != nil {
		return err

	}

	return os.WriteFile(filepath.Join(kratixDir, "output", "backstage", promise.GetName()+"-template.yaml"), templateBytes, 0777)
}

func generateBackstageTemplateWithoutProperties(rrCRD *v1.CustomResourceDefinition) (Template, error) {
	//Easier to generate from string than manually fill out go struct
	baseTemplate := []byte(fmt.Sprintf(`---
apiVersion: scaffolder.backstage.io/v1beta3
kind: Template
metadata:
  description: %[2]s as a Service
  name: %[1]s-promise-template
  tags:
  - syntasso
  - kratix
  - experimental
  title: %[2]s
spec:
  lifecycle: experimental
  owner: kratix-platform
  steps:
  - action: kubernetes:apply
    id: k-apply
    input:
      manifest: ""
      namespaced: true
    name: Create a %[1]s
  type: service`, rrCRD.Spec.Names.Kind, strings.Title(rrCRD.Spec.Names.Kind)))

	template := Template{}
	err := yamlsig.Unmarshal(baseTemplate, &template)
	if err != nil {
		return Template{}, fmt.Errorf("failed to unmarshal:"+string(baseTemplate)+": %w", err.Error())
	}
	return template, nil
}
