package finalize

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/libbuildpack"
)

type Logger interface {
	BeginStep(format string, args ...interface{})
	Info(format string, args ...interface{})
	Error(format string, args ...interface{})
	Protip(tip string, help_url string)
}
type Runner interface {
	Run(program string, args ...string) error
	CaptureStdout(program string, args ...string) (string, error)
}

type Finalize struct {
	BuildDir string
	Scripts  map[string]string
	Log      Logger
	Runner   Runner
}

func (c *Finalize) CreateDefaultEnv() {
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

func (c *Finalize) ListNodeConfig() {
	for _, env := range os.Environ() {
		if strings.HasPrefix(env, "NPM_CONFIG_") || strings.HasPrefix(env, "YARN_") || strings.HasPrefix(env, "NODE_") {
			c.Log.Info(env)
		}
	}

	if os.Getenv("NPM_CONFIG_PRODUCTION") == "true" && os.Getenv("NODE_ENV") != "production" {
		c.Log.Protip(
			"npm scripts will see NODE_ENV=production (not '"+os.Getenv("NODE_ENV")+"')",
			"https://docs.npmjs.com/misc/config#production")
	}
}

func (c *Finalize) BuildDependencies() error {
	// TODO run_if_present heroku-prebuild
	return chDir(c.BuildDir, func() error {
		if c.isYarn() {
			mirror_dir := filepath.Join(c.BuildDir, "npm-packages-offline-cache")
			if _, err := os.Stat(mirror_dir); err == nil {
				c.Log.Info("Found yarn mirror directory $mirror_dir")
				c.Log.Info("Running yarn in offline mode")
				if err := c.Runner.Run("yarn", "config", "set", "yarn-offline-mirror", mirror_dir); err != nil {
					return nil
				}
				return c.Runner.Run("yarn", "install", "--offline", "--pure-lockfile", "--ignore-engines", "--cache-folder", filepath.Join(c.BuildDir, ".cache", "yarn"))
			} else {
				c.Log.Info("Running yarn in online mode")
				c.Log.Protip("To run yarn in offline mode", "https://yarnpkg.com/blog/2016/11/24/offline-mirror")
				return c.Runner.Run("yarn", "install", "--pure-lockfile", "--ignore-engines", "--cache-folder", filepath.Join(c.BuildDir, ".cache", "yarn"))
			}
			// TODO package uptodate check see https://github.com/dgodd/nodejs-buildpack/blob/master/lib/dependencies.sh#L55
		} else if c.HasNodeModules() {
			c.Log.Info("Prebuild detected (node_modules already exists)")
			c.Log.Info("Rebuilding any native modules")
			os.Setenv("NODE_DIR", filepath.Join(c.BuildDir, ".cloudfoundry", "node")) // TODO not the same as original, and also haven't we already done this?
			if err := c.Runner.Run("npm", "rebuild"); err != nil {
				c.Log.Error("running npm rebuild: ", err)
				return err
			}
			// TODO shrinkwrap see lib/dependencies.sh
			c.Log.Info("Installing any new modules (package.json)")
			return c.Runner.Run("npm", "install", "--unsafe-perm", "--userconfig", filepath.Join(c.BuildDir, ".npmrc"))
		} else {
			if exists, _ := libbuildpack.FileExists(filepath.Join(c.BuildDir, "package.json")); exists {
				// TODO shrinkwrap see lib/dependencies.sh
				c.Log.Info("Installing node modules (package.json)")
				if err := c.Runner.Run("npm", "install", "--unsafe-perm", "--userconfig", filepath.Join(c.BuildDir, ".npmrc")); err != nil {
					c.Log.Error("running npm install: ", err)
					return err
				}
				return nil
			} else {
				c.Log.Info("Skipping (no package.json)")
			}
		}
		return nil
	})
}

func (c *Finalize) HasNodeModules() bool {
	dir := filepath.Join(c.BuildDir, "node_modules")
	if infos, err := ioutil.ReadDir(dir); err == nil {
		for _, fi := range infos {
			if fi.IsDir() {
				return true
			}
		}
	}
	return false
}

func (c *Finalize) isYarn() bool {
	exists, _ := libbuildpack.FileExists(filepath.Join(c.BuildDir, "yarn.lock"))
	return exists
}

func (c *Finalize) SummarizeBuild() {
	if os.Getenv("NODE_VERBOSE") != "false" {
		chDir(c.BuildDir, func() error {
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

func (c *Finalize) WarnNoStart() {
	if exists, _ := libbuildpack.FileExists(filepath.Join(c.BuildDir, "Procfile")); exists {
		return
	}
	if c.Scripts["start"] != "" {
		return
	}
	if exists, _ := libbuildpack.FileExists(filepath.Join(c.BuildDir, "server.js")); exists {
		return
	}
	c.Log.Protip("This app may not specify any way to start a node process", "https://docs.cloudfoundry.org/buildpacks/node/node-tips.html#start")
}

func (c *Finalize) WarnUnmetDep() {
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
	defer os.Chdir(wd)

	if err := os.Chdir(dir); err != nil {
		return nil
	}

	return fn()
}
