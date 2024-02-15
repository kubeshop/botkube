package plugin

import (
	"os"
	"path"
)

// TmpDir represents temporary directory.
type TmpDir string

// Get returns temporary directory path.
func (t TmpDir) Get() (string, bool) {
	if t != "" {
		return string(t), true
	}

	depDir := os.Getenv(DependencyDirEnvName)
	if depDir != "" {
		return depDir, false
	}

	return path.Join(os.TempDir(), "bin"), true
}

// GetDirectory returns temporary directory.
func (t TmpDir) GetDirectory() string {
	dir, _ := t.Get()
	return dir
}
