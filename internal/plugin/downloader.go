package plugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/hashicorp/go-getter"
)

// downloadBinary downloads binary into specific destination.
func downloadBinary(ctx context.Context, destPath string, url URL, autoDetectFilename bool) error {
	dir, filename := filepath.Split(destPath)
	err := os.MkdirAll(dir, dirPerms)
	if err != nil {
		return fmt.Errorf("while creating directory %q where the binary should be stored: %w", dir, err)
	}

	pwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("while getting working directory: %w", err)
	}

	tmpDestPath := destPath + ".downloading"
	if stat, err := os.Stat(tmpDestPath); err == nil && stat.IsDir() {
		if err = os.RemoveAll(tmpDestPath); err != nil {
			return fmt.Errorf("while deleting temporary directory %q: %w", tmpDestPath, err)
		}
	}

	urlWithGoGetterMagicParams := fmt.Sprintf("%s?filename=%s", url.URL, filename)
	if url.Checksum != "" {
		urlWithGoGetterMagicParams = fmt.Sprintf("%s&checksum=%s", urlWithGoGetterMagicParams, url.Checksum)
	}

	getterCli := &getter.Client{
		Ctx:  ctx,
		Src:  urlWithGoGetterMagicParams,
		Dst:  tmpDestPath,
		Pwd:  pwd,
		Mode: getter.ClientModeAny,
	}

	err = getterCli.Get()
	if err != nil {
		return fmt.Errorf("while downloading binary from URL %q: %w", url, err)
	}

	if stat, err := os.Stat(tmpDestPath); err == nil && stat.IsDir() {
		if autoDetectFilename {
			filename, err = getFirstFileInDirectory(tmpDestPath)
			if err != nil {
				return fmt.Errorf("while getting binary name")
			}
		}

		tempFileName := filepath.Join(tmpDestPath, filename)

		if err = os.Rename(tempFileName, destPath); err != nil {
			return fmt.Errorf("while renaming binary %q: %w", tempFileName, err)
		}
		if err = os.RemoveAll(tmpDestPath); err != nil {
			return fmt.Errorf("while deleting temporary directory %q: %w", tmpDestPath, err)
		}
	}
	if stat, err := os.Stat(destPath); err == nil && !stat.IsDir() {
		err = os.Chmod(destPath, binPerms)
		if err != nil {
			return fmt.Errorf("while setting permissions for %q: %w", destPath, err)
		}
	}

	return nil
}

// getFirstFileInDirectory returns the first file that it finds in a given directory.
//
// We use go-getter's 'filename' parameter to rename downloaded asset into a given name. However, it works only for files,
// has no effect in directory mode.
//
// Consider such download URL:
//
//	http://example.com/executor_kubectl_darwin_amd64.tar.gz?filename=executor_v0.13.0_kubectl
//
// because it is an archive the 'filename' param is ignored, and it is unpacked to 'executor_kubectl_darwin_amd64' directory
// and even there is just one file, it is not renamed.
// As a result, we cannot use the 'filename' value for doing 'os.Rename', so we use that function to get the real filename.
func getFirstFileInDirectory(dir string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		return e.Name(), nil
	}
	return "", nil
}
