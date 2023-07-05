package getter

import (
	"context"
	"fmt"
	"os"

	"github.com/hashicorp/go-getter"
)

// Download downloads data from a given source to local file system under a given destination path.
func Download(ctx context.Context, src, dst string) error {
	pwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("while getting current dir: %w", err)
	}

	// Build the client
	client := &getter.Client{
		Ctx:  ctx,
		Src:  src,
		Dst:  dst,
		Pwd:  pwd,
		Mode: getter.ClientModeDir,
	}

	return client.Get()
}
