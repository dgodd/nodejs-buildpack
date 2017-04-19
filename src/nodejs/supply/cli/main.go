package main

import (
	"nodejs/packagejson"
	"nodejs/supply"
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack"
)

func main() {
	buildDir := os.Args[1]
	cacheDir := os.Args[2]
	depsDir := os.Args[3]
	depIndex := os.Args[4]

	compiler, err := libbuildpack.NewCompiler([]string{buildDir, cacheDir, "", depsDir}, libbuildpack.Log)
	if err != nil {
		compiler.Log.BeginStep("Build failed")
		os.Exit(10)
	}

	if err := compiler.CheckBuildpackValid(); err != nil {
		compiler.Log.BeginStep("Build failed")
		os.Exit(11)
	}

	json := libbuildpack.NewJSON()
	packageJson, err := packagejson.New(compiler.Log, json, compiler.BuildDir)
	if err != nil {
		compiler.Log.BeginStep("Build failed")
		os.Exit(12)
	}

	supplier := &supply.Supply{
		BuildDir: compiler.BuildDir,
		DepDir:   filepath.Join(compiler.DepsDir, depIndex),
		Engines:  packageJson.Engines,
		Manifest: compiler.Manifest,
		Log:      compiler.Log,
		Runner:   compiler.Command,
	}

	if err := supplier.InstallBins(); err != nil {
		compiler.Log.BeginStep("Build failed")
		os.Exit(13)
	}
}
