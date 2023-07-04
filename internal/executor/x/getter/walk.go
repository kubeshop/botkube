// Package getter.
//
// Code copied from: https://github.com/facebookarchive/symwalk/blob/42004b9f322246749dd73ad71008b1f3160c0052/walk.go#L12-L45
// BSD License
//
// # For symwalk software
//
// Copyright (c) 2015, Facebook, Inc. All rights reserved.
package getter

import (
	"io/fs"
	"os"
	"path/filepath"
)

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
