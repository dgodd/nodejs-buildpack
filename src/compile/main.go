package main

import (
	"compile/cache"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack"
)

func main() {
	compiler, err := libbuildpack.NewCompiler(os.Args[1:], libbuildpack.Log)
	if err != nil {
		os.Exit(10)
	}

	if err := compiler.CheckBuildpackValid(); err != nil {
		os.Exit(11)
	}

	if err := compiler.LoadSuppliedDeps(); err != nil {
		compiler.Log.Error("LoadSupplied Deps failed")
		os.Exit(12)
	}

	runner := libbuildpack.NewCommandRunner()
	runner.SetOutput(libbuildpack.Log.GetOutput())

	cacher := &cache.Cache{
		BuildDir: compiler.BuildDir,
		CacheDir: compiler.CacheDir,
		Logger:   libbuildpack.Log,
		Runner:   runner,
	}

	c := &NodejsCompiler{Compiler: compiler, JSON: libbuildpack.NewJSON(), Runner: runner}
	if err = compile(c, cacher); err != nil {
		compiler.Log.Error("Compile failed")
		os.Exit(13)
	}

	compiler.StagingComplete()
}

func compile(c *NodejsCompiler, cacher *cache.Cache) error {
	if c.hasNodeModules() {
		c.Compiler.Log.Protip("It is recommended to vendor the application's Node.js dependencies", "http://docs.cloudfoundry.org/buildpacks/node/index.html#vendoring")
	}

	var packageJson PackageJson
	if err := packageJson.Load(c.Compiler.Log, c.JSON, c.Compiler.BuildDir); err != nil {
		return err
	}

	// warn_prebuilt_modules() // TODO: I DISAGREE AND AM IGNORING UNTIL DISAGREED WITH

	c.Compiler.Log.BeginStep("Creating runtime environment")
	libbuildpack.WriteProfileD(c.Compiler.BuildDir, "nodejs.sh", PROFILE_NODEJS)
	// export_env_dir() ignored due to supply above
	c.CreateDefaultEnv()
	c.ListNodeConfig()

	// TODO: Should this live elsewhere?
	os.Setenv("PATH", filepath.Join(c.Compiler.BuildDir, ".cloudfoundry", "node", "bin")+":"+os.Getenv("PATH"))
	os.Setenv("PATH", filepath.Join(c.Compiler.BuildDir, ".cloudfoundry", "yarn", "bin")+":"+os.Getenv("PATH"))

	c.Compiler.Log.BeginStep("Installing binaries")
	if err := c.InstallBins(packageJson.Engines); err != nil {
		return err
	}

	c.Compiler.Log.BeginStep("Restoring cache")
	cacher.Restore()

	c.Compiler.Log.BeginStep("Building dependencies")
	if err := c.BuildDependencies(); err != nil {
		return err
	}

	c.Compiler.Log.BeginStep("Caching build")
	cacher.Save()

	c.Compiler.Log.BeginStep("Build succeeded!")
	c.summarizeBuild()
	c.warnNoStart(packageJson.Scripts)
	c.warnUnmetDep()

	return nil
}
