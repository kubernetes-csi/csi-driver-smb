package embedded

import (
	"os"
	"testing"

	"github.com/kubernetes-csi/csi-test/pkg/sanity"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

func TestMyDriverGinkgo(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "CSI Sanity Test Suite")
}

// The test suite into which the sanity tests get embedded may already
// have before/after suite functions. There can only be one such
// function. Here we define empty ones because then Ginkgo
// will start complaining at runtime when invoking the embedded case
// in hack/e2e.sh if a PR adds back such functions in the sanity test
// code.
var _ = BeforeSuite(func() {})
var _ = AfterSuite(func() {})

var _ = Describe("MyCSIDriver", func() {
	Context("Config A", func() {
		config := &sanity.Config{
			TargetPath:  os.TempDir() + "/csi",
			StagingPath: os.TempDir() + "/csi",
			Address:     "/tmp/e2e-csi-sanity.sock",
		}

		BeforeEach(func() {})

		AfterEach(func() {})

		Describe("CSI Driver Test Suite", func() {
			sanity.GinkgoTest(config)
		})
	})
})
