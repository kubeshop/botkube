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

// symwalkFunc calls the provided WalkFn for regular files.
// However, when it encounters a symbolic link, it resolves the link fully using the
// filepath.EvalSymlinks function and recursively calls symwalk.Walk on the resolved path.
// This ensures that unlink filepath.Walk, traversal does not stop at symbolic links.
//
// Note that symwalk.Walk does not terminate if there are any non-terminating loops in
// the file structure.
func walk(filename string, linkDirname string, walkFn fs.WalkDirFunc) error {
	return filepath.WalkDir(filename, func(path string, d fs.DirEntry, err error) error {
		if fname, err := filepath.Rel(filename, path); err == nil {
			path = filepath.Join(linkDirname, fname)
		} else {
			return err
		}

		if err == nil && d.Type()&os.ModeSymlink == os.ModeSymlink {
			finalPath, err := filepath.EvalSymlinks(path)
			if err != nil {
				return err
			}
			info, err := os.Lstat(finalPath)
			if err != nil {
				return walkFn(path, d, err)
			}
			if info.IsDir() {
				return walk(finalPath, path, walkFn)
			}
		}

		return walkFn(path, d, err)
	})
}

// Walk extends filepath.Walk to also follow symlinks
func Walk(path string, walkFn fs.WalkDirFunc) error {
	return walk(path, path, walkFn)
}
