package lib_test

import (
	"os"
	"path/filepath"

	"github.com/aclevername/kratix-backstage-generator-pipeline/lib"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Generator", func() {
	var kratixDir string
	BeforeEach(func() {
		var err error
		kratixDir, err = os.MkdirTemp(os.TempDir(), "generator")
		Expect(err).NotTo(HaveOccurred())
		os.MkdirAll(filepath.Join(kratixDir, "output"), 0777)
		os.MkdirAll(filepath.Join(kratixDir, "input"), 0777)
		os.MkdirAll(filepath.Join(kratixDir, "metadata"), 0777)
	})

	AfterEach(func() {
		Expect(os.RemoveAll(kratixDir)).To(Succeed())
	})

	Describe("resource workflows", func() {
		BeforeEach(func() {
			err := os.WriteFile(filepath.Join(kratixDir, "input", "object.yaml"), []byte(`---
kind: Redis
apiVersion: marketplace.kratix.io/v1alpha1
metadata:
  name: bob
  namespace: team-a`), 0777)
			Expect(err).NotTo(HaveOccurred())
		})

		It("generates a valid component", func() {
			Expect(lib.Generate(kratixDir, "resource", "redis")).To(Succeed())

			fileContent, err := os.ReadFile(filepath.Join(kratixDir, "metadata", "destination-selectors.yaml"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(fileContent)).To(Equal(`- directory: backstage
  matchLabels:
    environment: backstage`))

			fileContent, err = os.ReadFile(filepath.Join(kratixDir, "output", "backstage", "bob-component.yaml"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(fileContent)).To(Equal(`---
apiVersion: backstage.io/v1alpha1
kind: Component
metadata:
  name: redis-bob
  title: "redis bob"
  description: bob created via redis Promise
  annotations:
    backstage.io/kubernetes-label-selector: redis-cr=bob
  links:
  - url: https://github.com/syntasso/kratix-backstage
    title: Support
    icon: help
spec:
  type: service
  lifecycle: production
  owner: kratix-worker
  dependsOn:
    - component:default/redis
  providesApis:
    - namespace-server-api
`))
		})
	})

	Describe("simple promise workflows", func() {
		BeforeEach(func() {
			err := os.WriteFile(filepath.Join(kratixDir, "input", "object.yaml"), []byte(`---
apiVersion: platform.kratix.io/v1alpha1
kind: Promise
metadata:
  name: redis
spec:
  api:
    apiVersion: apiextensions.k8s.io/v1
    kind: CustomResourceDefinition
    metadata:
      name: redis.marketplace.kratix.io
    spec:
      group: marketplace.kratix.io
      names:
        kind: redis
        plural: redis
        singular: redis
      scope: Namespaced
      versions:
        - name: v1alpha1
          schema:
            openAPIV3Schema:
              properties:
                spec:
                  properties:
                    size:
                      default: small
                      description: |
                        Size of this Redis deployment. If small, it deploy redis with a single replica; if large, deploy redis with 3 replicas.
                      pattern: ^(small|large)$
                      type: string
                  type: object
              type: object
          served: true
          storage: true`), 0777)
			Expect(err).NotTo(HaveOccurred())
		})

		It("generates a valid component", func() {
			Expect(lib.Generate(kratixDir, "promise", "redis")).To(Succeed())

			fileContent, err := os.ReadFile(filepath.Join(kratixDir, "metadata", "destination-selectors.yaml"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(fileContent)).To(Equal(`- directory: backstage
  matchLabels:
    environment: backstage`))

			fileContent, err = os.ReadFile(filepath.Join(kratixDir, "output", "backstage", "redis-component.yaml"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(fileContent)).To(Equal(`---
apiVersion: backstage.io/v1alpha1
kind: Component
metadata:
  annotations:
    backstage.io/kubernetes-id: redis
  description: Create a redis
  links:
  - icon: help
    title: Support
    url: https://github.com/syntasso/kratix-backstage
  name: redis
  title: redis Promise
spec:
  dependsOn:
  - component:default/kratix
  lifecycle: production
  owner: kratix-platform
  providesApis:
  - redis-promise-api
  type: promise
`))

			fileContent, err = os.ReadFile(filepath.Join(kratixDir, "output", "backstage", "redis-template.yaml"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(fileContent)).To(Equal(`apiVersion: scaffolder.backstage.io/v1beta3
kind: Template
metadata:
  creationTimestamp: null
  description: Redis as a Service
  name: redis-promise-template
  tags:
  - syntasso
  - kratix
  - experimental
  title: Redis
spec:
  lifecycle: experimental
  owner: kratix-platform
  parameters:
  - properties:
      name:
        description: Name for the request in the platform cluster
        title: Metadata.Name
        type: string
      namespace:
        description: Namespace for the request in the platform cluster
        title: Metadata.Namespace
        type: string
      size:
        description: |
          Size of this Redis deployment. If small, it deploy redis with a single replica; if large, deploy redis with 3 replicas.
        title: Spec.Size
        type: string
    required:
    - namespace
    - name
    title: Redis as a Service
  steps:
  - action: kubernetes:apply
    id: k-apply
    input:
      manifest: |
        apiVersion: marketplace.kratix.io/v1alpha1
        kind: redis
        metadata:
          creationTimestamp: null
          labels:
            backstage.io/kubernetes-id: redis
          name: ${{ parameters.name }}
          namespace: ${{ parameters.namespace}}
        spec:
          size: ${{ parameters.size }}
      namespaced: true
    name: Create a redis
  type: service
`))
		})
	})

	FDescribe("complex promise workflows", func() {
		BeforeEach(func() {
			err := os.WriteFile(filepath.Join(kratixDir, "input", "object.yaml"), []byte(`---
apiVersion: platform.kratix.io/v1alpha1
kind: Promise
metadata:
  name: redis
spec:
  api:
    apiVersion: apiextensions.k8s.io/v1
    kind: CustomResourceDefinition
    metadata:
      name: redis.marketplace.kratix.io
    spec:
      group: marketplace.kratix.io
      names:
        kind: redis
        plural: redis
        singular: redis
      scope: Namespaced
      versions:
      - name: v1alpha1
        schema:
          openAPIV3Schema:
            properties:
              spec:
                properties:
                  env:
                    default: dev
                    type: string
                  plugins:
                    type: array
                    default: []
                    description: Plugins to install in the requested Redis
                    items:
                      description: Plugin defines a single Redis plugin.
                      properties:
                        downloadURL:
                          description: DownloadURL is the custom url from where plugin
                            has to be downloaded.
                          type: string
                        name:
                          description: Name is the name of Redis plugin
                          type: string
                        version:
                          description: Version is the version of Redis plugin
                          type: string
                      required:
                      - name
                      - version
                      type: object
                type: object
            type: object
        served: true
        storage: true`), 0777)
			Expect(err).NotTo(HaveOccurred())
		})

		It("generates a valid component", func() {
			Expect(lib.Generate(kratixDir, "promise", "redis")).To(Succeed())

			fileContent, err := os.ReadFile(filepath.Join(kratixDir, "output", "backstage", "redis-template.yaml"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(fileContent)).To(Equal(`apiVersion: scaffolder.backstage.io/v1beta3
kind: Template
metadata:
  creationTimestamp: null
  description: Redis as a Service
  name: redis-promise-template
  tags:
  - syntasso
  - kratix
  - experimental
  title: Redis
spec:
  lifecycle: experimental
  owner: kratix-platform
  parameters:
  - properties:
      name:
        description: Name for the request in the platform cluster
        title: Metadata.Name
        type: string
      namespace:
        description: Namespace for the request in the platform cluster
        title: Metadata.Namespace
        type: string
      env:
        title: Spec.Env
        type: string
      plugins:
			  title: Spec.Plugins
        type: array
        ui:options:
          addable: true
          orderable: true
          removable: true
        items:
          type: object
          properties:
            downloadURL:
              title: Spec.Plugins[].DownloadURL
              type: string
              ui:widget: radio
            name:
              title: Spec.Plugins[].Name
              type: string
            version:
              title: Spec.Plugins[].Version
              type: string
			plugins:
    required:
    - namespace
    - name
    title: Redis as a Service
  steps:
  - action: kubernetes:apply
    id: k-apply
    input:
      manifest: |
        apiVersion: marketplace.kratix.io/v1alpha1
        kind: redis
        metadata:
          creationTimestamp: null
          labels:
            backstage.io/kubernetes-id: redis
          name: ${{ parameters.name }}
          namespace: ${{ parameters.namespace}}
        spec:
          size: ${{ parameters.size }}
      namespaced: true
    name: Create a redis
  type: service
`))
		})
	})
})
