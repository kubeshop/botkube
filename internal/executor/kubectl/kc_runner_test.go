package kubectl

import (
	"context"

	"github.com/kubeshop/botkube/pkg/pluginx"
)

type executeFn func(ctx context.Context, rawCmd string, mutators ...pluginx.ExecuteCommandMutation) (pluginx.ExecuteCommandOutput, error)

func NewMockedBinaryRunner(mock executeFn) *BinaryRunner {
	return &BinaryRunner{
		executeCommand: mock,
	}
}
