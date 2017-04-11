package cache

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"

	"github.com/cloudfoundry/libbuildpack"
)

type Runner interface {
	Run(program string, args ...string) error
	CaptureStdout(program string, args ...string) (string, error)
}
type Logger interface {
	Info(format string, args ...interface{})
	Error(format string, args ...interface{})
}

type Cache struct {
	BuildDir string
	CacheDir string
	Getenv   func(string) string
	Logger   Logger
	Runner   Runner
}

func (c *Cache) Save() error {
	c.Logger.Info("Clearing previous node cache")
	if err := c.Runner.Run("rm", "-rf", filepath.Join(c.CacheDir, "node")); err != nil {
		return err
	}
	if c.Getenv("NODE_MODULES_CACHE") == "false" {
		c.Logger.Info("Skipping cache save (disabled by config)")
		return nil
	}

	if err := c.saveSignature(); err != nil {
		return err
	}

	dirs, dirSource := c.cacheDirectories()
	c.Logger.Info("Saving %d cacheDirectories (%s):", len(dirs), dirSource)
	for _, dir := range dirs {
		if exists, _ := libbuildpack.FileExists(filepath.Join(c.BuildDir, dir)); exists {
			c.Logger.Info("- %s", dir)
			c.Runner.Run("cp", "-a", filepath.Join(c.BuildDir, dir), filepath.Join(c.CacheDir, "node"))
		} else {
			c.Logger.Info("- %s (nothing to cache)", dir)
		}
	}

	return c.removeFromSlug()
}

func (c *Cache) Restore() error {
	cacheStatus := c.getCacheStatus()
	if cacheStatus == "valid" {
		dirs, dirSource := c.cacheDirectories()
		c.Logger.Info("Loading %d from cacheDirectories (%s):", len(dirs), dirSource)
		for _, dir := range dirs {
			if exists, _ := libbuildpack.FileExists(filepath.Join(c.BuildDir, dir)); exists {
				c.Logger.Info("- %s (exists - skipping)", dir)
			} else if exists, _ := libbuildpack.FileExists(filepath.Join(c.CacheDir, "node", dir)); exists {
				c.Logger.Info("- %s", dir)
				os.MkdirAll(filepath.Dir(filepath.Join(c.BuildDir, "node", dir)), 0755)
				c.Runner.Run("mv", filepath.Join(c.CacheDir, "node", dir), filepath.Join(c.BuildDir, "node", dir))
			} else {
				c.Logger.Info("- %s (not cached - skipping)", dir)
			}
		}
	} else {
		c.Logger.Info("Skipping cache restore (%s)", cacheStatus)
	}
	return nil
}

func (c *Cache) removeFromSlug() error {
	if err := os.RemoveAll(filepath.Join(c.BuildDir, ".npm")); err != nil {
		return err
	}
	if err := os.RemoveAll(filepath.Join(c.BuildDir, ".cache", "yarn")); err != nil {
		return err
	}
	return nil
}

func (c *Cache) cacheDirectories() ([]string, string) {
	var data map[string][]string
	libbuildpack.NewJSON().Load(filepath.Join(c.BuildDir, "package.json"), &data)
	if len(data["cacheDirectories"]) > 0 {
		return data["cacheDirectories"], "package.json"
	} else if len(data["cache_directories"]) > 0 {
		return data["cache_directories"], "package.json"
	}
	return []string{".npm", ".cache/yarn", "bower_components"}, "default"
}

func (c *Cache) saveSignature() error {
	if err := os.MkdirAll(filepath.Join(c.CacheDir, "node"), 0755); err != nil {
		return err
	}
	return ioutil.WriteFile(filepath.Join(c.CacheDir, "node", "signature"), []byte(c.signature()), 0644)
}

func (c *Cache) signature() string {
	nodeVersion, _ := c.Runner.CaptureStdout("node", "--version")
	npmVersion, _ := c.Runner.CaptureStdout("npm", "--version")
	yarnVersion, _ := c.Runner.CaptureStdout("yarn", "--version")
	return strings.TrimSpace(nodeVersion) + "; " + strings.TrimSpace(npmVersion) + "; " + strings.TrimSpace(yarnVersion)
}

func (c *Cache) getCacheStatus() string {
	if c.Getenv("NODE_MODULES_CACHE") == "false" {
		return "disabled by config"
	}
	currentSignature, err := ioutil.ReadFile(filepath.Join(c.CacheDir, "node", "signature"))
	if err == nil && string(currentSignature) == c.signature() {
		return "valid"
	}
	return "new runtime signature"
}
