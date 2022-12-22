package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const specialConfigFileNamePrefix = "_"

// FileSystemProvider allows consumer to pass config files statically
type FileSystemProvider struct {
	Files []string
}

// NewFileSystemProvider initializes new static config source provider
func NewFileSystemProvider(configs []string) *FileSystemProvider {
	return &FileSystemProvider{Files: configs}
}

// Configs returns list of config file locations.
func (e *FileSystemProvider) Configs(_ context.Context) (YAMLFiles, error) {
	configPaths := sortCfgFiles(e.Files)

	var out YAMLFiles
	for _, path := range configPaths {
		raw, err := os.ReadFile(filepath.Clean(path))
		if err != nil {
			return nil, fmt.Errorf("while reading a file: %w", err)
		}
		out = append(out, raw)
	}

	return out, nil
}

// sortCfgFiles sorts the config files so that the files that has specialConfigFileNamePrefix are moved to the end of the slice.
func sortCfgFiles(paths []string) []string {
	var ordinaryCfgFiles []string
	var specialCfgFiles []string
	for _, path := range paths {
		_, filename := filepath.Split(path)

		if strings.HasPrefix(filename, specialConfigFileNamePrefix) {
			specialCfgFiles = append(specialCfgFiles, path)
			continue
		}

		ordinaryCfgFiles = append(ordinaryCfgFiles, path)
	}

	return append(ordinaryCfgFiles, specialCfgFiles...)
}
