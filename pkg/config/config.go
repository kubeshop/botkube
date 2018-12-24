package config

import (
	"gopkg.in/yaml.v2"
	"io/ioutil"
	"os"
	"path/filepath"
)

var configFileName = "kubeopsconfig.yaml"

type Config struct {
	Resources       []Resource
	Rbac            []Resource
	Recommendations bool
	Communications  Communications
	Events          K8SEvents
}

type K8SEvents struct {
	Types []string
	//	Kinds []Resource
}

type Resource struct {
	Name       string
	Namespaces []string
	Events     []string
}

type Namespaces struct {
	Namespaces []string `json:"namespaces"`
}

type Communications struct {
	Slack Slack
}

type Slack struct {
	Channel string
	Token   string
}

// New returns new Config
func New() (*Config, error) {
	c := &Config{}
	configPath := os.Getenv("KUBEOPS_CONFIG_PATH")
	configFile := filepath.Join(configPath, configFileName)
	file, err := os.Open(configFile)
	if err != nil {
		return c, err
	}

	b, err := ioutil.ReadAll(file)
	if err != nil {
		return c, err
	}

	if len(b) != 0 {
		yaml.Unmarshal(b, c)
	}
	return c, nil
}
