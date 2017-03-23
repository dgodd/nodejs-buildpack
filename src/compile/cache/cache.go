package cache

import "path/filepath"

type Runner interface {
	Run(program string, args ...string) error
}
type Logger interface {
	Info(format string, args ...interface{})
	Error(format string, args ...interface{})
}

type Cache struct {
	BuildDir string
	CacheDir string
	Logger   Logger
	Runner   Runner
}

func (c *Cache) Save() error {
	c.Logger.Error("TODO")
	return nil
}

func (c *Cache) Restore() error {
	c.Logger.Error("TODO")
	return nil
}

func (c *Cache) RemoveFromSlug() error {
	if err := c.Runner.Run("rm", "-rf", filepath.Join(c.BuildDir, ".npm")); err != nil {
		return err
	}
	if err := c.Runner.Run("rm", "-rf", filepath.Join(c.BuildDir, ".cache", "yarn")); err != nil {
		return err
	}
	return nil
}
