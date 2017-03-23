package cache_test

import (
	"compile/cache"
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//go:generate mockgen -source=cache.go -destination=mocks_cache_test.go --package=cache_test

var _ = Describe("Cache", func() {
	var (
		buildDir   string
		cacheDir   string
		err        error
		mockCtrl   *gomock.Controller
		mockLogger *MockLogger
		mockRunner *MockRunner
		subject    *cache.Cache
	)

	BeforeEach(func() {
		buildDir, err = ioutil.TempDir("", "nodejs-buildpack.build.")
		Expect(err).To(BeNil())

		cacheDir, err = ioutil.TempDir("", "nodejs-buildpack.cache.")
		Expect(err).To(BeNil())

		mockCtrl = gomock.NewController(GinkgoT())
		mockLogger = NewMockLogger(mockCtrl)
		mockRunner = NewMockRunner(mockCtrl)
	})

	JustBeforeEach(func() {
		subject = &cache.Cache{BuildDir: buildDir, CacheDir: cacheDir, Logger: mockLogger, Runner: mockRunner}
	})

	AfterEach(func() {
		mockCtrl.Finish()

		err = os.RemoveAll(buildDir)
		Expect(err).To(BeNil())

		err = os.RemoveAll(cacheDir)
		Expect(err).To(BeNil())
	})

	Describe("Save", func() {
		PIt("...", func() {})
	})
	Describe("Restore", func() {
		PIt("...", func() {})
	})
	Describe("RemoveFromSlug", func() {
		It("removes cache dirs from slug", func() {
			mockRunner.EXPECT().Run("rm", "-rf", filepath.Join(buildDir, ".npm")).Return(nil)
			mockRunner.EXPECT().Run("rm", "-rf", filepath.Join(buildDir, ".cache", "yarn")).Return(nil)
			err := subject.RemoveFromSlug()
			Expect(err).To(BeNil())
		})
		It("Passes up errors", func() {
			expectedErr := errors.New("error from rm")
			mockRunner.EXPECT().Run("rm", "-rf", filepath.Join(buildDir, ".npm")).Return(expectedErr)
			err := subject.RemoveFromSlug()
			Expect(err).To(Equal(expectedErr))
		})
	})
})
