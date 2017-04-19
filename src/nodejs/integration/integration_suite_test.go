package integration_test

import (
	"io/ioutil"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

var buildpackVersion string

func TestIntegration(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Integration Suite")
}

var _ = BeforeSuite(func() {
	data, err := ioutil.ReadFile("../../../VERSION")
	Expect(err).NotTo(HaveOccurred())
	buildpackVersion = string(data)
})
