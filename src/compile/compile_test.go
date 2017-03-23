package main_test

import (
	c "compile"
	"io/ioutil"
	"os"

	"bytes"

	"github.com/cloudfoundry/libbuildpack"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

//go:generate mockgen -source=vendor/github.com/cloudfoundry/libbuildpack/manifest.go --destination=mocks_manifest_test.go --package=main_test --imports=.=github.com/cloudfoundry/libbuildpack

var _ = Describe("Compile", func() {
	var (
		buildDir     string
		cacheDir     string
		depsDir      string
		nc           *c.NodejsCompiler
		logger       libbuildpack.Logger
		buffer       *bytes.Buffer
		err          error
		mockCtrl     *gomock.Controller
		mockManifest *MockManifest
	)

	BeforeEach(func() {
		buildDir, err = ioutil.TempDir("", "nodejs-buildpack.build.")
		Expect(err).To(BeNil())

		cacheDir, err = ioutil.TempDir("", "nodejs-buildpack.cache.")
		Expect(err).To(BeNil())

		depsDir = ""

		buffer = new(bytes.Buffer)

		logger = libbuildpack.NewLogger()
		logger.SetOutput(buffer)

		mockCtrl = gomock.NewController(GinkgoT())
		mockManifest = NewMockManifest(mockCtrl)
	})

	JustBeforeEach(func() {
		bpc := &libbuildpack.Compiler{BuildDir: buildDir,
			CacheDir: cacheDir,
			DepsDir:  depsDir,
			Manifest: mockManifest,
			Log:      logger}

		nc = &c.NodejsCompiler{Compiler: bpc}
	})

	AfterEach(func() {
		err = os.RemoveAll(buildDir)
		Expect(err).To(BeNil())

		err = os.RemoveAll(cacheDir)
		Expect(err).To(BeNil())
	})

	// Describe("InstallGo", func() {
	// 	var (
	// 		oldGoRoot    string
	// 		oldPath      string
	// 		goInstallDir string
	// 		dep          libbuildpack.Dependency
	// 	)

	// 	BeforeEach(func() {
	// 		oldPath = os.Getenv("PATH")
	// 		oldPath = os.Getenv("GOROOT")
	// 		goInstallDir = filepath.Join(cacheDir, "go1.3.4")

	// 		dep = libbuildpack.Dependency{Name: "go", Version: "1.3.4"}
	// 		mockManifest.EXPECT().InstallDependency(dep, goInstallDir).Return(nil).Times(1)
	// 	})

	// 	AfterEach(func() {
	// 		err = os.Setenv("PATH", oldPath)
	// 		Expect(err).To(BeNil())

	// 		err = os.Setenv("GOROOT", oldGoRoot)
	// 		Expect(err).To(BeNil())
	// 	})

	// 	It("Creates a bin directory", func() {
	// 		err = gc.InstallGo("1.3.4")
	// 		Expect(err).To(BeNil())

	// 		Expect(filepath.Join(buildDir, "bin")).To(BeADirectory())
	// 	})

	// 	It("Sets up GOROOT", func() {
	// 		err = gc.InstallGo("1.3.4")
	// 		Expect(err).To(BeNil())

	// 		Expect(os.Getenv("GOROOT")).To(Equal(filepath.Join(goInstallDir, "go")))
	// 	})

	// 	It("adds go to the PATH", func() {
	// 		err = gc.InstallGo("1.3.4")
	// 		Expect(err).To(BeNil())

	// 		newPath := fmt.Sprintf("%s:%s", oldPath, filepath.Join(goInstallDir, "go", "bin"))
	// 		Expect(os.Getenv("PATH")).To(Equal(newPath))
	// 	})

	// 	Context("go is already cached", func() {
	// 		BeforeEach(func() {
	// 			mockManifest.EXPECT().InstallDependency(dep, goInstallDir).Times(0)
	// 			err = os.MkdirAll(filepath.Join(goInstallDir, "go"), 0755)
	// 			Expect(err).To(BeNil())
	// 		})

	// 		It("uses the cached version", func() {
	// 			err = gc.InstallGo("1.3.4")
	// 			Expect(err).To(BeNil())

	// 			Expect(buffer.String()).To(ContainSubstring("-----> Using go1.3.4"))
	// 		})
	// 	})

	// 	Context("go is not already cached", func() {
	// 		BeforeEach(func() {
	// 			err = os.MkdirAll(filepath.Join(cacheDir, "go4.3.2", "go"), 0755)
	// 			Expect(err).To(BeNil())
	// 		})

	// 		It("installs go", func() {
	// 			err = gc.InstallGo("1.3.4")
	// 			Expect(err).To(BeNil())

	// 			Expect(buffer.String()).To(ContainSubstring("-----> Installing go1.3.4"))
	// 		})

	// 		It("clears the cache", func() {
	// 			err = gc.InstallGo("1.3.4")
	// 			Expect(err).To(BeNil())

	// 			Expect(filepath.Join(cacheDir, "go4.3.2", "go")).NotTo(BeADirectory())
	// 		})

	// 		It("creates the install directory", func() {
	// 			err = gc.InstallGo("1.3.4")
	// 			Expect(err).To(BeNil())

	// 			Expect(filepath.Join(cacheDir, "go1.3.4")).To(BeADirectory())
	// 		})
	// 	})

	// })
})
