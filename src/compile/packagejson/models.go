package packagejson

import (
	"os"
	"path/filepath"

	"github.com/cloudfoundry/libbuildpack"
)

type Engines struct {
	Node string
	Iojs string
	Npm  string
	Yarn string
}
type PackageJson struct {
	Engines Engines
	Scripts map[string]string
}

func New(log libbuildpack.Logger, json libbuildpack.JSON, dir string) (*PackageJson, error) {
	packageJson := new(PackageJson)
	if _, err := os.Stat(filepath.Join(dir, "package.json")); os.IsNotExist(err) {
		log.Warning("No package.json found")
	} else {
		if err := json.Load(filepath.Join(dir, "package.json"), &packageJson); err != nil {
			log.Error("Unable to parse package.json")
			return nil, err
		}
	}
	return packageJson, nil
}
