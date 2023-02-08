package kubectl

import (
	"context"
)

type executeFn func(ctx context.Context, rawCmd string, envs map[string]string) (string, error)

func NewMockedBinaryRunner(mock executeFn) *BinaryRunner {
	return &BinaryRunner{
		executeCommandWithEnvs: mock,
	}
}
