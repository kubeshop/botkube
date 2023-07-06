package getter

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"os"
	"path/filepath"
)

var hasher = sha256.New()

func sha(in string) string {
	hasher.Reset()
	hasher.Write([]byte(in))
	return base64.URLEncoding.EncodeToString(hasher.Sum(nil))
}

// EnsureDownloaded downloads given sources only if not yet downloaded.
// It's a weak comparison based on the source path.
func EnsureDownloaded(ctx context.Context, templateSources []Source, dir string) error {
	for _, tpl := range templateSources {
		dst := filepath.Join(dir, sha(tpl.Ref))
		err := runIfFileDoesNotExist(dst, func() error {
			return Download(ctx, tpl.Ref, dst)
		})
		if err != nil {
			return err
		}
	}

	return nil
}

func runIfFileDoesNotExist(path string, fn func() error) error {
	_, err := os.Stat(path)
	switch {
	case err == nil:
	case os.IsNotExist(err):
		return fn()
	default:
		return err
	}
	return nil
}
