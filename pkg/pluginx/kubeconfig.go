package pluginx

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
)

func PersistKubeConfig(ctx context.Context, kc []byte) (string, func(context.Context) error, error) {
	if len(kc) == 0 {
		return "", nil, fmt.Errorf("received empty kube config")
	}

	file, err := os.CreateTemp("", "kubeconfig-")
	if err != nil {
		return "", nil, err
	}

	abs, err := filepath.Abs(file.Name())
	if err != nil {
		return "", nil, err
	}

	if _, err = file.Write(kc); err != nil {
		return "", nil, err
	}

	deleteFn := func(context.Context) error {
		return os.RemoveAll(abs)
	}

	return abs, deleteFn, nil
}
