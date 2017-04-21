package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/cloudfoundry/libbuildpack"
)

type Manifest struct {
	Dependencies []struct {
		Name    string `yaml:"name"`
		Version string `yaml:"version"`
		Uri     string `yaml:"uri"`
		Md5     string `yaml:"md5"`
	} `yaml:"dependencies"`
}

func main() {
	var m Manifest
	err := libbuildpack.NewYAML().Load("manifest.yml", &m)
	if err != nil {
		panic(err)
	}

	manifest, err := libbuildpack.NewManifest(".", time.Now())
	if err != nil {
		panic(err)
	}

	if err := os.MkdirAll("deps", 0755); err != nil {
		panic(err)
	}

	r := strings.NewReplacer("/", "_", ":", "_", "?", "_", "&", "_")
	for _, dep := range m.Dependencies {
		path := filepath.Join("deps", r.Replace(dep.Uri))
		fmt.Println(dep.Uri)
		fmt.Println(path)
		if err := manifest.FetchDependency(libbuildpack.Dependency{Name: dep.Name, Version: dep.Version}, path); err != nil {
			panic(err)
		}
	}

}
