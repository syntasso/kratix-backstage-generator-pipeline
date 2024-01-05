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
	Title             string   `json:"title"`
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
	Title      string                `json:"title"`
}

type Properties struct {
	Description string                `json:"description,omitempty"`
	Title       string                `json:"title"`
	Type        string                `json:"type,omitempty"`
	Items       *Item                 `json:"items,omitempty"`
	Properties  map[string]Properties `json:"properties,omitempty"`
}

type Item struct {
	Type       string                `json:"type,omitempty"`
	Properties map[string]Properties `json:"properties,omitempty"`
}

type RR struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              string `json:"spec,omitempty"`
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
			Name:      "${{ parameters.objname }}",
			Namespace: "${{ parameters.objnamespace}}",
			Labels: map[string]string{
				"backstage.io/kubernetes-id": rrCRD.Spec.Names.Kind,
			},
		},
		Spec: "${{ parameters.spec | dump }}",
	}

	//Generate the parameter properties based on the CRD
	objMetaProps := map[string]Properties{}

	objMetaProps["objnamespace"] = Properties{
		Description: "Namespace for the request in the platform cluster",
		Title:       "Namespace",
		Type:        "string",
	}

	objMetaProps["objname"] = Properties{
		Description: "Name for the request in the platform cluster",
		Title:       "Name",
		Type:        "string",
	}

	template.Spec.Parameters = []Parameter{
		{
			Properties: objMetaProps,
			Title:      strings.Title(rrCRD.Spec.Names.Kind) + " Instance Metadata",
		},
	}

	for key, prop := range rrCRD.Spec.Versions[0].Schema.OpenAPIV3Schema.Properties["spec"].Properties {
		specFieldProp := map[string]Properties{}
		specFieldProp["spec"] = Properties{
			Type:       "object",
			Properties: map[string]Properties{},
			Title:      "",
		}

		if prop.XPreserveUnknownFields == nil || !*prop.XPreserveUnknownFields {
			specFieldProp["spec"].Properties[key] = genProperties("", key, prop)
			template.Spec.Parameters = append(template.Spec.Parameters, Parameter{
				Properties: specFieldProp,
				Title:      strings.Title(key),
			})
		}
	}

	sampleRRBytes, err := yamlsig.Marshal(rrManifestTemplate)
	if err != nil {
		return err

	}

	template.Spec.Steps[0].Input.Manifest = string(sampleRRBytes)

	//Convert to bytes
	templateBytes, err := yamlsig.Marshal(template)
	if err != nil {
		return err

	}

	os.Remove("../test-debugging-output")
	os.WriteFile("../test-debugging-output", templateBytes, 0777)

	return os.WriteFile(filepath.Join(kratixDir, "output", "backstage", promise.GetName()+"-template.yaml"), templateBytes, 0777)
}

func genProperties(prefix, key string, prop v1.JSONSchemaProps) Properties {
	p := Properties{
		Description: prop.Description,
		Title:       prefix + strings.Title(key),
	}
	p.Type = prop.Type

	switch p.Type {
	case "array":
		p.Items = &Item{
			Properties: map[string]Properties{},
			Type:       "object",
		}
		for subKey, subProp := range prop.Items.Schema.Properties {
			if subProp.XPreserveUnknownFields == nil || !*subProp.XPreserveUnknownFields {
				p.Items.Properties[subKey] = genProperties((p.Title + "."), subKey, subProp)
			}
		}
	case "object":
		if len(prop.Properties) > 0 {
			p.Properties = map[string]Properties{}
			for subKey, subProp := range prop.Properties {
				if subProp.XPreserveUnknownFields == nil || !*subProp.XPreserveUnknownFields {
					p.Properties[subKey] = genProperties((p.Title + "."), subKey, subProp)
				}
			}
		}
	}

	return p
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
