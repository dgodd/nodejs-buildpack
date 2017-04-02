package supply_test

import (
	"compile/packagejson"
	"compile/supply"
	"io/ioutil"
	"os"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//go:generate mockgen -source=supply.go -destination=mocks_supply_test.go --package=supply_test

var _ = Describe("Supply", func() {
	var (
		buildDir string
		depDir   string
		err      error

		engines      packagejson.Engines
		mockCtrl     *gomock.Controller
		mockManifest *MockManifest
		mockLogger   *MockLogger
		mockRunner   *MockRunner
		subject      *supply.Supply
	)

	BeforeEach(func() {
		buildDir, err = ioutil.TempDir("", "nodejs-buildpack.build.")
		Expect(err).To(BeNil())

		depDir, err = ioutil.TempDir("", "nodejs-buildpack.dep.")
		Expect(err).To(BeNil())

		mockCtrl = gomock.NewController(GinkgoT())
		mockManifest = NewMockManifest(mockCtrl)
		mockLogger = NewMockLogger(mockCtrl)
		mockRunner = NewMockRunner(mockCtrl)
	})

	JustBeforeEach(func() {
		subject = &supply.Supply{BuildDir: buildDir, DepDir: depDir, Engines: engines, Manifest: mockManifest, Log: mockLogger, Runner: mockRunner}
	})

	AfterEach(func() {
		mockCtrl.Finish()

		err = os.RemoveAll(buildDir)
		Expect(err).To(BeNil())

		err = os.RemoveAll(depDir)
		Expect(err).To(BeNil())
	})

	Describe("CreateDefaultEnv", func() {
		// It("removes cache dirs from slug", func() {
		// 	mockRunner.EXPECT().Run("rm", "-rf", filepath.Join(buildDir, ".npm")).Return(nil)
		// 	mockRunner.EXPECT().Run("rm", "-rf", filepath.Join(buildDir, ".cache", "yarn")).Return(nil)
		// 	err := subject.RemoveFromSlug()
		// 	Expect(err).To(BeNil())
		// })
		// It("Passes up errors", func() {
		// 	expectedErr := errors.New("error from rm")
		// 	mockRunner.EXPECT().Run("rm", "-rf", filepath.Join(buildDir, ".npm")).Return(expectedErr)
		// 	err := subject.RemoveFromSlug()
		// 	Expect(err).To(Equal(expectedErr))
		// })
	})
})
