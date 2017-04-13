package integration_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/tidwall/gjson"
)

type cfConfig struct {
	SpaceFields struct {
		GUID string
	}
}
type cfApps struct {
	Resources []struct {
		Metadata struct {
			GUID string `json:"guid"`
		} `json:"metadata"`
	} `json:"resources"`
}
type cfInstance struct {
	State string `json:"state"`
}

type App struct {
	Name      string
	Buildpack string
	Stdout    *bytes.Buffer
	appGUID   string
}

func (a *App) SpaceGUID() (string, error) {
	bytes, err := ioutil.ReadFile(filepath.Join(os.Getenv("HOME"), ".cf", "config.json"))
	if err != nil {
		return "", err
	}
	var config cfConfig
	if err := json.Unmarshal(bytes, &config); err != nil {
		return "", err
	}
	return config.SpaceFields.GUID, nil
}

func (a *App) AppGUID() (string, error) {
	if a.appGUID != "" {
		return a.appGUID, nil
	}
	guid, err := a.SpaceGUID()
	if err != nil {
		return "", err
	}
	cmd := exec.Command("cf", "curl", "/v2/apps?q=space_guid:"+guid+"&q=name:"+a.Name)
	bytes, err := cmd.Output()
	if err != nil {
		return "", err
	}
	var apps cfApps
	if err := json.Unmarshal(bytes, &apps); err != nil {
		return "", err
	}
	if len(apps.Resources) != 1 {
		return "", fmt.Errorf("Expected one app, found %d", len(apps.Resources))
	}
	a.appGUID = apps.Resources[0].Metadata.GUID
	return a.appGUID, nil
}

func (a *App) InstanceStates() ([]string, error) {
	guid, err := a.AppGUID()
	if err != nil {
		return []string{}, err
	}
	cmd := exec.Command("cf", "curl", "/v2/apps/"+guid+"/instances")
	bytes, err := cmd.Output()
	if err != nil {
		return []string{}, err
	}
	var data map[string]cfInstance
	if err := json.Unmarshal(bytes, &data); err != nil {
		return []string{}, err
	}
	var states []string
	for _, value := range data {
		states = append(states, value.State)
	}
	return states, nil
}

func (a *App) Push() error {
	command := exec.Command("cf", "push", a.Name, "--no-start", "--random-route", "-b", a.Buildpack, "-p", filepath.Join("../../../cf_spec/fixtures", a.Name))
	if data, err := command.Output(); err != nil {
		fmt.Println(string(data))
		return err
	}

	command = exec.Command("cf", "logs", a.Name)
	a.Stdout = bytes.NewBuffer(nil)
	command.Stdout = a.Stdout
	if err := command.Start(); err != nil {
		return err
	}

	command = exec.Command("cf", "start", a.Name)
	if data, err := command.Output(); err != nil {
		fmt.Println(string(data))
		return err
	}
	return nil
}

func (a *App) GetUrl(path string) (string, error) {
	guid, err := a.AppGUID()
	if err != nil {
		return "", err
	}
	cmd := exec.Command("cf", "curl", "/v2/apps/"+guid+"/summary")
	data, err := cmd.Output()
	if err != nil {
		return "", err
	}
	host := gjson.Get(string(data), "routes.0.host").String()
	domain := gjson.Get(string(data), "routes.0.domain.name").String()
	return fmt.Sprintf("http://%s.%s%s", host, domain, path), nil
}

func (a *App) GetBody(path string) (string, error) {
	url, err := a.GetUrl(path)
	if err != nil {
		return "", err
	}
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	// TODO: Non 200 ??
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(data), err
}

func (a *App) Destroy() error {
	command := exec.Command("cf", "delete", "-f", a.Name)
	if err := command.Run(); err != nil {
		return err
	}
	return nil
}

var _ = Describe("Deploy", func() {
	const buildpack = "https://github.com/dgodd/nodejs-buildpack.git#golang"
	// const buildpack = "nodejs_buildpack"
	var app *App
	AfterEach(func() {
		if app != nil {
			app.Destroy()
		}
		app = nil
	})

	Context("when specifying a range for the nodeJS version in the package.json", func() {
		BeforeEach(func() {
			app = &App{Name: "node_version_range", Buildpack: buildpack, appGUID: ""}
		})
		It("resolves to a nodeJS version successfully", func() {
			Expect(app.Push()).To(Succeed())
			Expect(app.InstanceStates()).To(Equal([]string{"RUNNING"}))
			Expect(app.Stdout.String()).To(ContainSubstring("Installing node 4."))

			Expect(app.GetBody("/")).To(Equal("Hello, World!"))
		})
	})

	Context("when specifying a version 6 for the nodeJS version in the package.json", func() {
		BeforeEach(func() {
			app = &App{Name: "node_version_6", Buildpack: buildpack, appGUID: ""}
		})
		It("resolves to a nodeJS version successfully", func() {
			Expect(app.Push()).To(Succeed())
			Expect(app.InstanceStates()).To(Equal([]string{"RUNNING"}))
			Expect(app.Stdout.String()).To(ContainSubstring("Installing node 6."))

			Expect(app.GetBody("/")).To(Equal("Hello, World!"))
		})
	})

	Context("when not specifying a nodeJS version in the package.json", func() {
		BeforeEach(func() {
			app = &App{Name: "without_node_version", Buildpack: buildpack, appGUID: ""}
		})
		It("resolves to the stable nodeJS version successfully", func() {
			Expect(app.Push()).To(Succeed())
			Expect(app.InstanceStates()).To(Equal([]string{"RUNNING"}))
			Expect(app.Stdout.String()).To(ContainSubstring("Installing node 4."))

			Expect(app.GetBody("/")).To(Equal("Hello, World!"))

			Specify("correctly displays the buildpack version", func() {
				Expect(app.Stdout.String()).To(ContainSubstring("node.js " + buildpackVersion))
			})
		})
	})

	Context("with an unreleased nodejs version", func() {
		BeforeEach(func() {
			app = &App{Name: "unreleased_node_version", Buildpack: buildpack, appGUID: ""}
		})

		It("displays a nice error messages and gracefully fails", func() {
			Expect(app.Push()).ToNot(Succeed())
			Expect(app.Stdout.String()).To(ContainSubstring("Installing node 9000.0.0"))
			Expect(app.Stdout.String()).To(ContainSubstring("DEPENDENCY MISSING IN MANIFEST:"))
			Expect(app.Stdout.String()).To(ContainSubstring("-----> Build failed"))
		})
	})

	Context("with an unsupported, but released, nodejs version", func() {
		BeforeEach(func() {
			app = &App{Name: "unsupported_node_version", Buildpack: buildpack, appGUID: ""}
		})

		It("displays a nice error messages and gracefully fails", func() {
			Expect(app.Push()).ToNot(Succeed())
			Expect(app.Stdout.String()).To(ContainSubstring("Installing node 4.1.1"))
			Expect(app.Stdout.String()).To(ContainSubstring("DEPENDENCY MISSING IN MANIFEST:"))
			Expect(app.Stdout.String()).To(ContainSubstring("-----> Build failed"))
		})
	})

	PContext("with an app that has vendored dependencies", func() {})
	PContext("with an app with a yarn.lock file", func() {})
	PContext("with an app with a yarn.lock and vendored dependencies", func() {})

	Context("with an app with an out of date yarn.lock", func() {
		BeforeEach(func() {
			app = &App{Name: "out_of_date_yarn_lock", Buildpack: buildpack, appGUID: ""}
		})

		It("warns that yarn.lock is out of date", func() {
			Expect(app.Push()).ToNot(Succeed())
			Expect(app.Stdout.String()).To(ContainSubstring("yarn.lock is outdated"))
			Expect(app.InstanceStates()).To(Equal([]string{"RUNNING"}))
		})
	})

	PContext("with an app with no vendored dependencies", func() {})

	Context("with an incomplete node_modules directory", func() {
		BeforeEach(func() {
			app = &App{Name: "incomplete_node_modules", Buildpack: buildpack, appGUID: ""}
		})

		PIt("downloads missing dependencies from package.json", func() {
			Expect(app.Push()).ToNot(Succeed())
			Expect(app.InstanceStates()).To(Equal([]string{"RUNNING"}))

			// expect(Dir).to_not exist("cf_spec/fixtures/node_web_app_with_incomplete_node_modules/node_modules/hashish")
			// expect(app).to have_file("/app/node_modules/hashish")
			// expect(app).to have_file("/app/node_modules/express")
		})
	})

	PContext("with an incomplete package.json", func() {})
	PContext("with a cached buildpack in an air gapped environment", func() {})
})
