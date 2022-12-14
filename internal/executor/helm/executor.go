package helm

import (
	"context"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/api/executor"
)

var _ executor.Executor = &Executor{}

type Executor struct {}

func NewExecutor() *Executor {
	return &Executor{}
}

func (Executor) Metadata(context.Context) (api.MetadataOutput, error) {
	return api.MetadataOutput{
		Version:     "version",
		Description: "Echo is an example Botkube executor plugin used during e2e tests. It's not meant for production usage.",
	}, nil
}


// install, uninstall, upgrade, rollback, list, version, test, status
//   --repo - custom flag which adds the repo for a short period of time
//   support --set for now
//   ensure multiline commands work properly

// Execute returns a given command as response.
func (Executor) Execute(_ context.Context, in executor.ExecuteInput) (executor.ExecuteOutput, error) {

	finalCfg, err := MergeConfigs(in.Configs)
	if err != nil {
		return executor.ExecuteOutput{}, fmt.Errorf("while merging input configs: %w", err)
	}



	cmd := in.Command
	if finalCfg.ChangeResponseToUpperCase != nil && *finalCfg.ChangeResponseToUpperCase {
		data = strings.ToUpper(data)
	}

	return executor.ExecuteOutput{
		Data: data,
	}, nil
}
