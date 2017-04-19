package supply

import (
	"errors"
	"nodejs/packagejson"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/libbuildpack"
)

type Logger interface {
	Info(format string, args ...interface{})
	Error(format string, args ...interface{})
	Protip(tip string, help_url string)
}
type Runner interface {
	Run(program string, args ...string) error
	CaptureStdout(program string, args ...string) (string, error)
}
type Manifest interface {
	DefaultVersion(depName string) (libbuildpack.Dependency, error)
	InstallDependency(dep libbuildpack.Dependency, outputDir string) error
	AllDependencyVersions(string) []string
}

type Supply struct {
	BuildDir string
	DepDir   string
	Engines  packagejson.Engines
	Manifest Manifest
	Log      Logger
	Runner   Runner
}

func (c *Supply) InstallBins() error {
	if c.Engines.Iojs != "" {
		// TODO ; This was previously not true, old buildpack installed iojs from the internet
		c.Log.Error("io.js has merged with the Node.js project, please use that instead")
		return errors.New("io.js is not available")
	}

	c.Log.Info("engines.node (package.json):  " + c.Engines.Node) // TODO "unspecified" see https://github.com/dgodd/nodejs-buildpack/blob/master/bin/compile#L99
	c.Log.Info("engines.npm (package.json):   " + c.Engines.Npm)  // TODO "unspecified (use default)" see https://github.com/dgodd/nodejs-buildpack/blob/master/bin/compile#L100
	c.warnNodeEngine()

	if err := os.MkdirAll(filepath.Join(c.DepDir, "bin"), 0755); err != nil {
		return err
	}
	os.Setenv("PATH", filepath.Join(c.DepDir, "bin")+":"+os.Getenv("PATH"))

	if err := c.InstallNodejs(); err != nil {
		c.Log.Error("Failed to install node")
		return err
	}

	if err := c.InstallNpm(); err != nil {
		c.Log.Error("Failed to install npm: %v", err)
		return err
	}

	if c.isYarn() {
		if err := c.InstallYarn(); err != nil {
			c.Log.Error("Failed to install yarn: %v", err)
			return err
		}
	}

	return nil
}

func (c *Supply) InstallNodejs() error {
	version := c.Engines.Node
	if version == "" {
		dep, err := c.Manifest.DefaultVersion("node")
		if err != nil {
			return err
		}
		version = dep.Version
	} else {
		versionConstraint := version
		versions := c.Manifest.AllDependencyVersions("node")
		if matchingVersion, err := libbuildpack.FindMatchingVersion(versionConstraint, versions); err == nil {
			version = matchingVersion
		}
	}

	dep := libbuildpack.Dependency{Name: "node", Version: version}
	if err := c.Manifest.InstallDependency(dep, c.DepDir); err != nil {
		return err
	}
	if err := rename(c.DepDir, "node-v*", "node"); err != nil {
		return err
	}
	if err := os.Symlink(filepath.Join("..", "node", "bin", "npm"), filepath.Join(c.DepDir, "bin", "npm")); err != nil {
		return err
	}
	return os.Symlink(filepath.Join("..", "node", "bin", "node"), filepath.Join(c.DepDir, "bin", "node"))
}

func (c *Supply) InstallNpm() error {
	version := c.Engines.Npm
	npmVersion, err := c.Runner.CaptureStdout("npm", "--version")
	if err != nil {
		return err
	}
	npmVersion = strings.Trim(npmVersion, " \n")

	if version == "" {
		c.Log.Info("Using default version: %s", npmVersion)
	} else if npmVersion == version {
		c.Log.Info("npm %s already installed with node", npmVersion)
	} else {
		c.Log.Info("Downloading and installing npm %s (replacing version %s)...", version, npmVersion)
		_, err := c.Runner.CaptureStdout("npm", "install", "--unsafe-perm", "--quiet", "-g", "npm@"+version)
		if err != nil {
			c.Log.Error("We're unable to download the version of npm you've provided (%s).\nPlease remove the npm version specification in package.json", version)
			return err
		}
	}
	return nil
}

func (c *Supply) InstallYarn() error {
	version := c.Engines.Yarn
	tmpDir := filepath.Join(c.DepDir, "yarn_temp")
	dir := filepath.Join(c.DepDir, "yarn")
	if version == "" {
		dep, err := c.Manifest.DefaultVersion("yarn")
		if err != nil {
			return err
		}
		version = dep.Version
	} else {
		versionConstraint := version
		versions := c.Manifest.AllDependencyVersions("yarn")
		if matchingVersion, err := libbuildpack.FindMatchingVersion(versionConstraint, versions); err == nil {
			version = matchingVersion
		}
	}


	if err := c.Manifest.InstallDependency(libbuildpack.Dependency{Name: "yarn", Version: version}, tmpDir); err != nil {
		return err
	}
	if err := os.Rename(filepath.Join(tmpDir, "dist"), dir); err != nil {
		return err
	}
	if err := os.Chmod(filepath.Join(dir, "bin", "yarn"), 0755); err != nil {
		return err
	}
	return os.Symlink(filepath.Join("..", "yarn", "bin", "yarn"), filepath.Join(c.DepDir, "bin", "yarn"))
}

func (c *Supply) isYarn() bool {
	exists, _ := libbuildpack.FileExists(filepath.Join(c.BuildDir, "yarn.lock"))
	return exists
}

func (c *Supply) warnNodeEngine() {
	nodeEngine := c.Engines.Node
	if nodeEngine == "" {
		c.Log.Protip("Node version not specified in package.json", "http://docs.cloudfoundry.org/buildpacks/node/node-tips.html")
	} else if nodeEngine == "*" {
		c.Log.Protip("Dangerous semver range (*) in engines.node", "http://docs.cloudfoundry.org/buildpacks/node/node-tips.html")
	} else if nodeEngine[0] == '>' {
		c.Log.Protip("Dangerous semver range (>) in engines.node", "http://docs.cloudfoundry.org/buildpacks/node/node-tips.html")
	}
}

func rename(root, old, new string) error {
	globs, err := filepath.Glob(filepath.Join(root, old))
	if err != nil || len(globs) != 1 {
		return errors.New("Could not find " + old + " to rename")
	}
	return os.Rename(globs[0], filepath.Join(root, new))
}
