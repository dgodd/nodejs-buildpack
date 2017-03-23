package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/libbuildpack"
)

type NodejsCompiler struct {
	Compiler *libbuildpack.Compiler
	JSON     libbuildpack.JSON
	Runner   libbuildpack.CommandRunner
}

func (c *NodejsCompiler) CreateDefaultEnv() {
	if _, found := os.LookupEnv("NPM_CONFIG_PRODUCTION"); !found {
		os.Setenv("NPM_CONFIG_PRODUCTION", "true")
	}
	if _, found := os.LookupEnv("NPM_CONFIG_LOGLEVEL"); !found {
		os.Setenv("NPM_CONFIG_LOGLEVEL", "error")
	}
	if _, found := os.LookupEnv("NODE_MODULES_CACHE"); !found {
		os.Setenv("NODE_MODULES_CACHE", "true")
	}
	if _, found := os.LookupEnv("NODE_ENV"); !found {
		os.Setenv("NODE_ENV", "production")
	}
	if _, found := os.LookupEnv("NODE_VERBOSE"); !found {
		os.Setenv("NODE_VERBOSE", "false")
	}
}

func (c *NodejsCompiler) ListNodeConfig() {
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "NPM_CONFIG_") || strings.HasPrefix(env, "YARN_") || strings.HasPrefix(env, "NODE_") {
			c.Compiler.Log.Info(env)
		}
	}

	if os.Getenv("NPM_CONFIG_PRODUCTION") == "true" && os.Getenv("NODE_ENV") != "production" {
		c.Compiler.Log.Protip(
			"npm scripts will see NODE_ENV=production (not '"+os.Getenv("NODE_ENV")+"')",
			"https://docs.npmjs.com/misc/config#production")
	}
}

func (c *NodejsCompiler) InstallBins(engines Engines) error {
	if engines.Iojs != "" {
		// TODO ; This was previously not true, old buildpack installed iojs from the internet
		c.Compiler.Log.Error("io.js has merged with the Node.js project, please use that instead")
		return errors.New("io.js is not available")
	}

	c.Compiler.Log.Info("engines.node (package.json):  " + engines.Node) // TODO "unspecified" see https://github.com/dgodd/nodejs-buildpack/blob/master/bin/compile#L99
	c.Compiler.Log.Info("engines.npm (package.json):   " + engines.Npm)  // TODO "unspecified (use default)" see https://github.com/dgodd/nodejs-buildpack/blob/master/bin/compile#L100
	WarnNodeEngine(engines.Node, c.Compiler.Log)
	if err := c.InstallNodejs(engines.Node); err != nil {
		c.Compiler.Log.Error("Failed to install node")
		return err
	}
	if err := c.InstallNpm(engines.Npm); err != nil {
		c.Compiler.Log.Error("Failed to install npm: %v", err)
		return err
	}

	if c.isYarn() {
		if err := c.InstallYarn(engines.Yarn); err != nil {
			c.Compiler.Log.Error("Failed to install yarn: %v", err)
			return err
		}
	}

	return nil
}

func (c *NodejsCompiler) InstallNodejs(version string) error {
	if version == "" {
		dep, err := c.Compiler.Manifest.DefaultVersion("node")
		if err != nil {
			return err
		}
		version = dep.Version
	}

	dir := filepath.Join(c.Compiler.BuildDir, ".cloudfoundry")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	dep := libbuildpack.Dependency{Name: "node", Version: version}
	if err := c.Compiler.Manifest.InstallDependency(dep, dir); err != nil {
		return err
	}
	return rename(filepath.Join(c.Compiler.BuildDir, ".cloudfoundry"), "node-v*", "node")
}

func (c *NodejsCompiler) InstallNpm(version string) error {
	npmVersion, err := c.Runner.CaptureStdout("npm", "--version")
	if err != nil {
		return err
	}
	npmVersion = strings.Trim(npmVersion, " \n")

	if version == "" {
		c.Compiler.Log.Info("Using default version: %s", npmVersion)
	} else if npmVersion == version {
		c.Compiler.Log.Info("npm %s already installed with node", npmVersion)
	} else {
		c.Compiler.Log.Info("Downloading and installing npm %s (replacing version %s)...", version, npmVersion)
		_, err := c.Runner.CaptureStdout("npm", "install", "--unsafe-perm", "--quiet", "-g", "npm@"+version)
		if err != nil {
			c.Compiler.Log.Error("We're unable to download the version of npm you've provided (%s).\nPlease remove the npm version specification in package.json", version)
			return err
		}
	}
	return nil
}

func (c *NodejsCompiler) InstallYarn(version string) error {
	tmpDir := filepath.Join(c.Compiler.BuildDir, ".cloudfoundry", "yarn_temp")
	dir := filepath.Join(c.Compiler.BuildDir, ".cloudfoundry", "yarn")
	if version == "" {
		dep, err := c.Compiler.Manifest.DefaultVersion("yarn")
		if err != nil {
			return err
		}
		version = dep.Version
	}
	if err := c.Compiler.Manifest.InstallDependency(libbuildpack.Dependency{Name: "yarn", Version: version}, tmpDir); err != nil {
		return err
	}
	if err := os.Rename(filepath.Join(tmpDir, "dist"), dir); err != nil {
		return err
	}
	return os.Chmod(filepath.Join(dir, "bin", "yarn"), 0755)
}

func (c *NodejsCompiler) BuildDependencies() error {
	// TODO run_if_present heroku-prebuild
	if c.isYarn() {
		mirror_dir := filepath.Join(c.Compiler.BuildDir, "npm-packages-offline-cache")
		if _, err := os.Stat(mirror_dir); err == nil {
			c.Compiler.Log.Info("Found yarn mirror directory $mirror_dir")
			c.Compiler.Log.Info("Running yarn in offline mode")
			return chDir(c.Compiler.BuildDir, func() error {
				if err := c.Runner.Run("yarn", "config", "set", "yarn-offline-mirror", mirror_dir); err != nil {
					return nil
				}
				return c.Runner.Run("yarn", "install", "--offline", "--pure-lockfile", "--ignore-engines", "--cache-folder", filepath.Join(c.Compiler.BuildDir, ".cache", "yarn"))
			})
		} else {
			c.Compiler.Log.Info("Running yarn in online mode")
			c.Compiler.Log.Protip("To run yarn in offline mode", "https://yarnpkg.com/blog/2016/11/24/offline-mirror")
			return chDir(c.Compiler.BuildDir, func() error {
				return c.Runner.Run("yarn", "install", "--pure-lockfile", "--ignore-engines", "--cache-folder", filepath.Join(c.Compiler.BuildDir, ".cache", "yarn"))
			})
		}
		// TODO package uptodate check see https://github.com/dgodd/nodejs-buildpack/blob/master/lib/dependencies.sh#L55
	} else if c.hasNodeModules() {
		c.Compiler.Log.Info("Prebuild detected (node_modules already exists)")
		_ = chDir(c.Compiler.BuildDir, func() error {
			c.Compiler.Log.Info("Rebuilding any native modules")
			os.Setenv("NODE_DIR", filepath.Join(c.Compiler.BuildDir, ".cloudfoundry", "node")) // TODO not the same as original, and also haven't we already done this?
			_ = c.Runner.Run("npm", "rebuild")
			// TODO shrinkwrap see lib/dependencies.sh
			c.Compiler.Log.Info("Installing any new modules (package.json)")
			return chDir(c.Compiler.BuildDir, func() error {
				return c.Runner.Run("npm", "install", "--unsafe-perm", "--userconfig", filepath.Join(c.Compiler.BuildDir, ".npmrc"))
			})
		})
	} else {
		if exists, _ := libbuildpack.FileExists(filepath.Join(c.Compiler.BuildDir, "package.json")); exists {
			// TODO shrinkwrap see lib/dependencies.sh
			c.Compiler.Log.Info("Installing node modules (package.json)")
			return chDir(c.Compiler.BuildDir, func() error {
				if err := c.Runner.Run("npm", "install", "--unsafe-perm", "--userconfig", filepath.Join(c.Compiler.BuildDir, ".npmrc")); err != nil {
					c.Compiler.Log.Error("running npm install: ", err)
					return err
				}
				return nil
			})
		} else {
			c.Compiler.Log.Info("Skipping (no package.json)")
		}
	}
	return nil
}

func (c *NodejsCompiler) hasNodeModules() bool {
	dir := filepath.Join(c.Compiler.BuildDir, "node_modules")
	if infos, err := ioutil.ReadDir(dir); err == nil {
		for _, fi := range infos {
			if fi.IsDir() {
				return true
			}
		}
	}
	return false
}

func (c *NodejsCompiler) isYarn() bool {
	exists, _ := libbuildpack.FileExists(filepath.Join(c.Compiler.BuildDir, "yarn.lock"))
	return exists
}

func (c *NodejsCompiler) summarizeBuild() {
	if os.Getenv("NODE_VERBOSE") != "" {
		chDir(c.Compiler.BuildDir, func() error {
			if c.isYarn() {
				if out, err := c.Runner.CaptureStdout("yarn", "list", "--depth=0"); err == nil {
					fmt.Println(out)
				}
			} else {
				if out, err := c.Runner.CaptureStdout("npm", "ls", "--depth=0"); err == nil {
					idx := strings.Index(out, "\n")
					fmt.Println(out[(idx + 1):])
				}
			}
			return nil
		})
	}
}
func (c *NodejsCompiler) warnNoStart(scripts map[string]string) {
	if exists, _ := libbuildpack.FileExists(filepath.Join(c.Compiler.BuildDir, "Procfile")); exists {
		return
	}
	if scripts["start"] != "" {
		return
	}
	if exists, _ := libbuildpack.FileExists(filepath.Join(c.Compiler.BuildDir, "server.js")); exists {
		return
	}
	c.Compiler.Log.Protip("This app may not specify any way to start a node process", "https://docs.cloudfoundry.org/buildpacks/node/node-tips.html#start")
}
func (c *NodejsCompiler) warnUnmetDep() {
	//// TODO
	// if grep -qi 'unmet dependency' "$log_file" || grep -qi 'unmet peer dependency' "$log_file"; then
	//   warn "Unmet dependencies don't fail npm install but may cause runtime issues" "https://github.com/npm/npm/issues/7494"
	// fi
}

func chDir(dir string, fn func() error) error {
	wd, err := os.Getwd()
	if err != nil {
		return nil
	}

	if err := os.Chdir(dir); err != nil {
		return nil
	}

	if err := fn(); err != nil {
		return nil
	}

	return os.Chdir(wd)
}

func rename(root, old, new string) error {
	globs, err := filepath.Glob(filepath.Join(root, old))
	if err != nil || len(globs) != 1 {
		return errors.New("Could not find " + old + " to rename")
	}
	return os.Rename(globs[0], filepath.Join(root, new))
}
