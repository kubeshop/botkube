package getter

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Source holds information about source location.
type Source struct {
	Ref string `yaml:"ref"`
}

// Load downloads defined sources and read them from the FS.
func Load[T any](ctx context.Context, tmpDir string, templateSources []Source) ([]T, error) {
	err := EnsureDownloaded(ctx, templateSources, tmpDir)
	if err != nil {
		return nil, err
	}

	var out []T
	err = Walk(tmpDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}

		if filepath.Ext(d.Name()) != ".yaml" {
			return nil
		}

		file, err := os.ReadFile(filepath.Clean(path))
		if err != nil {
			return err
		}

		var cfg struct {
			Templates []T `yaml:"templates"`
		}
		err = yaml.Unmarshal(file, &cfg)
		if err != nil {
			return err
		}
		out = append(out, cfg.Templates...)
		return nil
	})
	if err != nil {
		return nil, err
	}

	return out, nil
}
