package integration_test

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
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
	Name       string
	Buildpack  string
	LogSession *gexec.Session
	appGUID    string
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
	command := exec.Command("cf", "push", a.Name, "--no-start", "-b", a.Buildpack, "-p", filepath.Join("../../../cf_spec/fixtures", a.Name))
	session, err := gexec.Start(command, GinkgoWriter, GinkgoWriter)
	if err != nil {
		return err
	}
	session.Wait(30 * time.Second)

	command = exec.Command("cf", "logs", a.Name)
	a.LogSession, err = gexec.Start(command, GinkgoWriter, GinkgoWriter)
	if err != nil {
		return err
	}

	command = exec.Command("cf", "start", a.Name)
	err = command.Run()
	if err != nil {
		return err
	}
	return nil
}

var _ = Describe("Deploy", func() {
	var appName, buildpack string
	BeforeEach(func() {
		buildpack = "https://github.com/dgodd/nodejs-buildpack.git#golang"
		appName = "with_yarn"
	})

	It("run", func() {
		app := &App{Name: appName, Buildpack: buildpack, appGUID: ""}
		Expect(app.Push()).To(Succeed())
		Expect(app.InstanceStates()).To(Equal([]string{"RUNNING"}))
		Expect(app.LogSession.Out).To(ContainSubstring("Downloading and installing node 4."))
	})
})
