package pluginx

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

func PersistKubeConfig(ctx context.Context, kc []byte) (string, func(context.Context) error, error) {
	if len(kc) == 0 {
		return "", nil, fmt.Errorf("received empty kube config")
	}

	file, err := os.CreateTemp("", "kubeconfig-")
	if err != nil {
		return "", nil, errors.Wrap(err, "while writing kube config to file")
	}

	abs, err := filepath.Abs(file.Name())
	if err != nil {
		return "", nil, errors.Wrap(err, "while writing kube config to file")
	}

	if _, err = file.Write(kc); err != nil {
		return "", nil, errors.Wrap(err, "while writing kube config to file")
	}

	deleteFn := func(context.Context) error {
		return os.RemoveAll(abs)
	}

	return abs, deleteFn, nil
}
