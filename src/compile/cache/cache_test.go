package cache_test

import (
	"compile/cache"
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
		environ    map[string]string
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

		environ = make(map[string]string)

		mockCtrl = gomock.NewController(GinkgoT())
		mockLogger = NewMockLogger(mockCtrl)
		mockRunner = NewMockRunner(mockCtrl)
	})

	JustBeforeEach(func() {
		subject = &cache.Cache{
			BuildDir: buildDir,
			CacheDir: cacheDir,
			Getenv:   func(key string) string { return environ[key] },
			Logger:   mockLogger,
			Runner:   mockRunner}
	})

	AfterEach(func() {
		mockCtrl.Finish()

		err = os.RemoveAll(buildDir)
		Expect(err).To(BeNil())

		err = os.RemoveAll(cacheDir)
		Expect(err).To(BeNil())
	})

	Describe("Save", func() {
		BeforeEach(func() {
			mockRunner.EXPECT().CaptureStdout("node", "--version").Return("1.2.3", nil).AnyTimes()
			mockRunner.EXPECT().CaptureStdout("npm", "--version").Return("4.5.6", nil).AnyTimes()
			mockRunner.EXPECT().CaptureStdout("yarn", "--version").Return("7.8.9", nil).AnyTimes()
		})
		It("clears the cache", func() {
			mockLogger.EXPECT().Info("Clearing previous node cache")
			mockLogger.EXPECT().Info(gomock.Any(), gomock.Any(), gomock.Any())
			mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
			mockRunner.EXPECT().Run("rm", "-rf", filepath.Join(cacheDir, "node")).Return(nil)
			Expect(subject.Save()).To(Succeed())
		})

		Context("NODE_MODULES_CACHE == false", func() {
			BeforeEach(func() {
				environ["NODE_MODULES_CACHE"] = "false"
				mockLogger.EXPECT().Info("Clearing previous node cache")
				mockRunner.EXPECT().Run("rm", "-rf", filepath.Join(cacheDir, "node")).Return(nil)
			})
			It("Skips caching", func() {
				mockLogger.EXPECT().Info("Skipping cache save (disabled by config)")
				Expect(subject.Save()).To(Succeed())
			})
		})

		Context("NODE_MODULES_CACHE == true (same as untrue)", func() {
			BeforeEach(func() {
				environ["NODE_MODULES_CACHE"] = "true"
				mockLogger.EXPECT().Info("Clearing previous node cache")
				mockRunner.EXPECT().Run("rm", "-rf", filepath.Join(cacheDir, "node")).Return(nil)
			})
			Context("package.json has cacheDirectories", func() {
				BeforeEach(func() {
					ioutil.WriteFile(filepath.Join(buildDir, "package.json"), []byte(`{"cacheDirectories":["a","b"]}`), 0644)
				})
				It("saves those directories", func() {
					os.MkdirAll(filepath.Join(buildDir, "a"), 0755)
					os.MkdirAll(filepath.Join(buildDir, "b"), 0755)

					mockLogger.EXPECT().Info("Saving %d cacheDirectories (%s):", 2, "package.json")
					mockLogger.EXPECT().Info("- %s", "a")
					mockRunner.EXPECT().Run("cp", "-a", filepath.Join(buildDir, "a"), filepath.Join(cacheDir, "node"))
					mockLogger.EXPECT().Info("- %s", "b")
					mockRunner.EXPECT().Run("cp", "-a", filepath.Join(buildDir, "b"), filepath.Join(cacheDir, "node"))
					Expect(subject.Save()).To(Succeed())
				})

				It("skips non-existent directories", func() {
					os.MkdirAll(filepath.Join(buildDir, "a"), 0755)

					mockLogger.EXPECT().Info("Saving %d cacheDirectories (%s):", 2, "package.json")
					mockLogger.EXPECT().Info("- %s", "a")
					mockRunner.EXPECT().Run("cp", "-a", filepath.Join(buildDir, "a"), filepath.Join(cacheDir, "node"))
					mockLogger.EXPECT().Info("- %s (nothing to cache)", "b")
					Expect(subject.Save()).To(Succeed())
				})
			})
			Context("package.json has cache_directories", func() {
				BeforeEach(func() {
					ioutil.WriteFile(filepath.Join(buildDir, "package.json"), []byte(`{"cache_directories":["c","d"]}`), 0644)
				})
				It("saves those directories", func() {
					os.MkdirAll(filepath.Join(buildDir, "c"), 0755)
					os.MkdirAll(filepath.Join(buildDir, "d"), 0755)

					mockLogger.EXPECT().Info("Saving %d cacheDirectories (%s):", 2, "package.json")
					mockLogger.EXPECT().Info("- %s", "c")
					mockRunner.EXPECT().Run("cp", "-a", filepath.Join(buildDir, "c"), filepath.Join(cacheDir, "node"))
					mockLogger.EXPECT().Info("- %s", "d")
					mockRunner.EXPECT().Run("cp", "-a", filepath.Join(buildDir, "d"), filepath.Join(cacheDir, "node"))
					Expect(subject.Save()).To(Succeed())
				})

				It("skips non-existent directories", func() {
					os.MkdirAll(filepath.Join(buildDir, "d"), 0755)

					mockLogger.EXPECT().Info("Saving %d cacheDirectories (%s):", 2, "package.json")
					mockLogger.EXPECT().Info("- %s (nothing to cache)", "c")
					mockLogger.EXPECT().Info("- %s", "d")
					mockRunner.EXPECT().Run("cp", "-a", filepath.Join(buildDir, "d"), filepath.Join(cacheDir, "node"))
					Expect(subject.Save()).To(Succeed())
				})
			})
			Context("package.json has neither cacheDirectories or cache_directories", func() {
				It("caches default directories", func() {
					mockLogger.EXPECT().Info("Saving %d cacheDirectories (%s):", 3, "default")
					mockLogger.EXPECT().Info("- %s (nothing to cache)", ".npm")
					mockLogger.EXPECT().Info("- %s (nothing to cache)", ".cache/yarn")
					mockLogger.EXPECT().Info("- %s (nothing to cache)", "bower_components")
					Expect(subject.Save()).To(Succeed())
				})
			})
		})

		It("saves cache signature", func() {
			mockLogger.EXPECT().Info(gomock.Any()).AnyTimes()
			mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
			mockLogger.EXPECT().Info(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
			mockRunner.EXPECT().Run("rm", "-rf", filepath.Join(cacheDir, "node")).Return(nil)
			Expect(subject.Save()).To(Succeed())

			Expect(ioutil.ReadFile(filepath.Join(cacheDir, "node", "signature"))).To(Equal([]uint8("1.2.3; 4.5.6; 7.8.9")))
		})

		It("remove caches from slug", func() {
			mockLogger.EXPECT().Info(gomock.Any()).AnyTimes()
			mockLogger.EXPECT().Info(gomock.Any(), gomock.Any()).AnyTimes()
			mockLogger.EXPECT().Info(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
			mockRunner.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()
			mockRunner.EXPECT().Run(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).AnyTimes()

			os.MkdirAll(filepath.Join(buildDir, ".npm", "a", "b", "c"), 0755)
			os.MkdirAll(filepath.Join(buildDir, ".cache", "yarn", "a", "b", "c"), 0755)

			Expect(subject.Save()).To(Succeed())

			Expect(filepath.Join(buildDir, ".npm")).ToNot(BeADirectory())
			Expect(filepath.Join(buildDir, ".cache", "yarn")).ToNot(BeADirectory())
		})
	})
	Describe("Restore", func() {
		PIt("HANDLES cache signature true/false", func() {})
	})
})
