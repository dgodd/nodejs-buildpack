package main

import (
	"compile/cache"
	"compile/finalize"
	"compile/packagejson"
	"os"

	"github.com/cloudfoundry/libbuildpack"
)

func main() {
	buildDir := os.Args[1]
	cacheDir := os.Args[2]
	depsDir := os.Args[3]
	// depIdx := os.Args[4]

	compiler, err := libbuildpack.NewCompiler([]string{buildDir, cacheDir, "", depsDir}, libbuildpack.Log)
	if err != nil {
		os.Exit(10)
	}

	if err := compiler.CheckBuildpackValid(); err != nil {
		os.Exit(11)
	}

	packageJson, err := packagejson.New(compiler.Log, libbuildpack.NewJSON(), compiler.BuildDir)
	if err != nil {
		panic(err)
		os.Exit(12)
	}

	if err := compiler.LoadSuppliedDeps(); err != nil {
		compiler.Log.Error("LoadSupplied Deps failed")
		os.Exit(13)
	}

	runner := libbuildpack.NewCommandRunner()
	runner.SetOutput(libbuildpack.Log.GetOutput())

	cacher := &cache.Cache{
		BuildDir: compiler.BuildDir,
		CacheDir: compiler.CacheDir,
		Getenv:   func(key string) string { return os.Getenv(key) },
		Logger:   libbuildpack.Log,
		Runner:   runner,
	}

	c := &finalize.Finalize{
		BuildDir: compiler.BuildDir,
		Scripts:  packageJson.Scripts,
		Log:      libbuildpack.Log,
		Runner:   runner,
	}
	if err = run(c, cacher); err != nil {
		compiler.Log.Error("Compile failed")
		os.Exit(13)
	}

	compiler.StagingComplete()
}

func run(c *finalize.Finalize, cacher *cache.Cache) error {
	if c.HasNodeModules() {
		c.Log.Protip("It is recommended to vendor the application's Node.js dependencies", "http://docs.cloudfoundry.org/buildpacks/node/index.html#vendoring")
	}

	// warn_prebuilt_modules() // TODO: I DISAGREE AND AM IGNORING UNTIL DISAGREED WITH

	c.Log.BeginStep("Creating runtime environment")
	libbuildpack.WriteProfileD(c.BuildDir, "nodejs.sh", finalize.PROFILE_NODEJS)
	// export_env_dir() ignored due to supply above
	c.CreateDefaultEnv()
	c.ListNodeConfig()

	c.Log.BeginStep("Restoring cache")
	cacher.Restore()

	c.Log.BeginStep("Building dependencies")
	if err := c.BuildDependencies(); err != nil {
		return err
	}

	c.Log.BeginStep("Caching build")
	cacher.Save()

	c.Log.BeginStep("Build succeeded!")
	c.SummarizeBuild()
	c.WarnNoStart()
	c.WarnUnmetDep()

	return nil
}
