package plugin

import (
	"os"
	"path"
)

type TmpDir string

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

func (t TmpDir) GetDirectory() string {
	dir, _ := t.Get()
	return dir
}
