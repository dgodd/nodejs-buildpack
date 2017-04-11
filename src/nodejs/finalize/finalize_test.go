package finalize_test

import (
	"io/ioutil"
	"nodejs/finalize"
	"os"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//go:generate mockgen -source=finalize.go -destination=mocks_finalize_test.go --package=finalize_test

var _ = Describe("Finalize", func() {
	var (
		buildDir string
		err      error

		scripts    map[string]string
		mockCtrl   *gomock.Controller
		mockLogger *MockLogger
		mockRunner *MockRunner
		subject    *finalize.Finalize
	)

	BeforeEach(func() {
		buildDir, err = ioutil.TempDir("", "nodejs-buildpack.build.")
		Expect(err).To(BeNil())

		scripts = make(map[string]string)
		mockCtrl = gomock.NewController(GinkgoT())
		mockLogger = NewMockLogger(mockCtrl)
		mockRunner = NewMockRunner(mockCtrl)
	})

	JustBeforeEach(func() {
		subject = &finalize.Finalize{BuildDir: buildDir, Scripts: scripts, Log: mockLogger, Runner: mockRunner}
	})

	AfterEach(func() {
		mockCtrl.Finish()

		err = os.RemoveAll(buildDir)
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
