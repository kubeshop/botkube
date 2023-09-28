package main

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/alexflint/go-arg"
	"github.com/hashicorp/go-plugin"
	"github.com/sanity-io/litter"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"

	thmate "github.com/kubeshop/botkube/internal/executor/thread-mate"
	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/api/executor"
	"github.com/kubeshop/botkube/pkg/pluginx"
)

const pluginName = "thread-mate"

// version is set via ldflags by GoReleaser.
var version = "dev"

// ThreadMateExecutor implements the Botkube executor plugin interface.
type ThreadMateExecutor struct {
	once sync.Map
}

func NewThreadMateExecutor() *ThreadMateExecutor {
	return &ThreadMateExecutor{}
}

// Metadata returns details about plugin.
func (*ThreadMateExecutor) Metadata(context.Context) (api.MetadataOutput, error) {
	return api.MetadataOutput{
		Version:     version,
		Description: "Streamlines managing assignment for incidents or user support",
		JSONSchema: api.JSONSchema{
			Value: thmate.JSONSchema,
		},
	}, nil
}

func (t *ThreadMateExecutor) init(cfg thmate.Config, kubeconfig []byte) (*thmate.ThreadMate, error) {
	svc, ok := t.once.Load(cfg.RoundRobinGroupName)
	if ok {
		return svc.(*thmate.ThreadMate), nil
	}
	kubeConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeconfig)
	if err != nil {
		return nil, fmt.Errorf("while reading kube config. %w", err)
	}
	k8sCli, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("while creating K8s clientset: %w", err)
	}

	cfgDumper := thmate.NewConfigMapDumper(k8sCli)

	newSvc := thmate.New(cfg, cfgDumper)
	newSvc.Start()

	t.once.Store(cfg.RoundRobinGroupName, newSvc)
	return newSvc, nil
}

// Execute returns a given command as a response.
func (t *ThreadMateExecutor) Execute(ctx context.Context, in executor.ExecuteInput) (executor.ExecuteOutput, error) {
	if err := pluginx.ValidateKubeConfigProvided(pluginName, in.Context.KubeConfig); err != nil {
		return executor.ExecuteOutput{}, err
	}

	var cmd thmate.Commands
	err := pluginx.ParseCommand(pluginName, in.Command, &cmd)
	switch {
	case err == nil:
	case errors.Is(err, arg.ErrHelp):
		msg, _ := t.Help(ctx)
		return executor.ExecuteOutput{
			Message: msg,
		}, nil
	default:
		return executor.ExecuteOutput{}, fmt.Errorf("while parsing input command: %w", err)
	}

	cfg, err := thmate.MergeConfigs(in.Configs)
	if err != nil {
		return executor.ExecuteOutput{}, fmt.Errorf("while merging configuration: %w", err)
	}

	svc, err := t.init(cfg, in.Context.KubeConfig)
	if err != nil {
		return executor.ExecuteOutput{}, fmt.Errorf("while initializing service: %w", err)
	}

	switch {
	case cmd.Pick != nil:
		msgs := svc.Pick(cmd.Pick, in.Context.Message)
		litter.Config.HideZeroValues = true
		litter.Dump(msgs)
		return executor.ExecuteOutput{
			Messages: msgs,
		}, nil
	case cmd.Get != nil && cmd.Get.Activity != nil:
		return executor.ExecuteOutput{
			Message: svc.GetActivity(cmd.Get.Activity, in.Context.Message),
		}, nil
	case cmd.Resolve != nil:
		return executor.ExecuteOutput{
			Message: svc.Resolve(cmd.Resolve, in.Context.Message),
		}, nil
	case cmd.Takeover != nil:
		return executor.ExecuteOutput{
			Message: svc.Takeover(cmd.Takeover, in.Context.Message),
		}, nil
	case cmd.Export != nil:
		return executor.ExecuteOutput{
			Message: svc.Export(cmd.Export),
		}, nil
	default:
		return executor.ExecuteOutput{
			Message: api.NewPlaintextMessage("Command not supported", false),
		}, nil
	}
}

func (*ThreadMateExecutor) Help(context.Context) (api.Message, error) {
	btnBuilder := api.NewMessageButtonBuilder()
	return api.Message{
		Sections: []api.Section{
			{
				Base: api.Base{
					Header: "Streamlines managing assignment for incidents or user support",
				},
				Buttons: []api.Button{
					btnBuilder.ForCommandWithDescCmd("Pick a person", "thread-mate pick"),
					btnBuilder.ForCommandWithDescCmd("Get Activity", "thread-mate get activity"),
				},
			},
		},
	}, nil
}

func main() {
	executor.Serve(map[string]plugin.Plugin{
		pluginName: &executor.Plugin{
			Executor: NewThreadMateExecutor(),
		},
	})
}
