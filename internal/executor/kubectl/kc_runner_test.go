package kubectl

import (
	"context"

	"github.com/kubeshop/botkube/pkg/plugin"
)

type executeFn func(ctx context.Context, rawCmd string, mutators ...plugin.ExecuteCommandMutation) (plugin.ExecuteCommandOutput, error)

func NewMockedBinaryRunner(mock executeFn) *BinaryRunner {
	return &BinaryRunner{
		executeCommand: mock,
	}
}
