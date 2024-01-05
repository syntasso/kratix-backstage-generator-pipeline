package lib_test

import (
	"testing"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/format"
)

func TestLib(t *testing.T) {
	format.MaxLength = 0
	format.PrintContextObjects = true
	format.TruncateThreshold = 100000
	RegisterFailHandler(Fail)
	RunSpecs(t, "Lib Suite")
}
