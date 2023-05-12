package pluginx

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pkg/errors"
)

func PersistKubeConfig(_ context.Context, kc []byte) (string, func(context.Context) error, error) {
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

// CheckKubeConfigProvided returns an error if a given kubeconfig is empty or nil.
func CheckKubeConfigProvided(pluginName string, kubeconfig []byte) error {
	if len(kubeconfig) != 0 {
		return nil
	}
	return fmt.Errorf("The kubeconfig data is missing. Please make sure that you have specified a valid RBAC configuration for %q plugin. Learn more at https://docs.botkube.io/configuration/rbac.", pluginName)
}
