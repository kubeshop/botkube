package commands

import (
	"embed"
	"fmt"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/kubeshop/botkube/internal/executor/x/template"
)

//go:embed store
var f embed.FS

func LoadTemplates() ([]template.Template, error) {
	dirs, err := f.ReadDir("store")
	if err != nil {
		return nil, err
	}

	var templates []template.Template
	for _, d := range dirs {
		fmt.Println(d.Name())
		if d.IsDir() {
			continue
		}
		file, err := f.ReadFile(filepath.Join("store", d.Name()))
		if err != nil {
			return nil, err
		}

		var cfg struct {
			Templates []template.Template `yaml:"templates"`
		}
		err = yaml.Unmarshal(file, &cfg)
		if err != nil {
			return nil, fmt.Errorf("while unmarshaling file %q: %v", d.Name(), err)
		}

		templates = append(templates, cfg.Templates...)
	}

	return templates, nil
}
