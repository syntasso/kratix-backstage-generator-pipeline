package lib_test

import (
	"io"
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

	Describe("promise workflows", func() {
		BeforeEach(func() {
			err := copyFile(filepath.Join("assets", "promise.yaml"), filepath.Join(kratixDir, "input", "object.yaml"))
			Expect(err).NotTo(HaveOccurred())
		})

		It("generates a valid component", func() {
			Expect(lib.Generate(kratixDir, "promise", "jenkin")).To(Succeed())

			actualContent, err := os.ReadFile(filepath.Join(kratixDir, "output", "backstage", "jenkins-template.yaml"))
			Expect(err).NotTo(HaveOccurred())

			disiredContent, err := os.ReadFile(filepath.Join("assets", "desired-jenkins-template.yaml"))
			Expect(err).NotTo(HaveOccurred())

			Expect(string(actualContent)).To(Equal(string(disiredContent)))

			actualContent, err = os.ReadFile(filepath.Join(kratixDir, "output", "backstage", "jenkins-component.yaml"))
			Expect(err).NotTo(HaveOccurred())

			disiredContent, err = os.ReadFile(filepath.Join("assets", "desired-jenkins-component.yaml"))
			Expect(err).NotTo(HaveOccurred())

			Expect(string(actualContent)).To(Equal(string(disiredContent)))
		})
	})
})

// source: https://stackoverflow.com/questions/21060945/simple-way-to-copy-a-file
func copyFile(srcpath, dstpath string) (err error) {
	r, err := os.Open(srcpath)
	if err != nil {
		return err
	}
	defer r.Close() // ignore error: file was opened read-only.

	w, err := os.Create(dstpath)
	if err != nil {
		return err
	}

	defer func() {
		// Report the error from Close, if any,
		// but do so only if there isn't already
		// an outgoing error.
		if c := w.Close(); c != nil && err == nil {
			err = c
		}
	}()

	_, err = io.Copy(w, r)
	return err
}
