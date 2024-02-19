package executor

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/hashicorp/go-plugin"
	"github.com/slack-go/slack"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/mathx"
)

const maxMessageNumberForSingleCommandExecution = 15

// Executor defines the Botkube executor plugin functionality.
type Executor interface {
	Execute(context.Context, ExecuteInput) (ExecuteOutput, error)
	Metadata(ctx context.Context) (api.MetadataOutput, error)
	Help(context.Context) (api.Message, error)
}

type (
	// ExecuteInput holds the input of the Execute function.
	ExecuteInput struct {
		// Context holds execution context.
		Context ExecuteInputContext
		// Command holds the command to be executed.
		Command string
		// Configs is a list of Executor configurations specified by users.
		Configs []*Config
	}

	// ExecuteInputContext holds execution context.
	ExecuteInputContext struct {
		// IsInteractivitySupported is set to true only if communication platform supports interactive Messages.
		IsInteractivitySupported bool

		// KubeConfig is the slice of byte representation of kubeconfig file content
		KubeConfig []byte

		// SlackState represents modal state. It's available only if:
		//  - IsInteractivitySupported is set to true,
		//  - and interactive actions were used in the response Message.
		// This is an alpha feature and may change in the future.
		// Most likely, it will be generalized to support all communication platforms.
		SlackState *slack.BlockActionStates

		// Message details that triggered a given Executor.
		// Limitations:
		//   - It's available only for SocketSlack. In the future, it may be adopted across other platforms.
		Message Message

		IncomingWebhook IncomingWebhookDetailsContext
	}

	// IncomingWebhookDetailsContext holds source incoming webhook context.
	IncomingWebhookDetailsContext struct {
		BaseSourceURL string
	}

	// Message holds information about the message that triggered a given Executor.
	Message struct {
		Text string
		URL  string
		User User

		// ParentActivityID is the ID of the parent activity. If user follows with messages in a thread, this ID represents the originating message that started that thread.
		// Otherwise, it's the ID of the initial message.
		ParentActivityID string
	}

	// User represents the user that sent a message.
	User struct {
		// ID represents users identifier.
		Mention string
		// DisplayName represents user display name. It can be empty.
		DisplayName string
	}

	// ExecuteOutput holds the output of the Execute function.
	ExecuteOutput struct {
		// Message represents the output of processing a given input command.
		// You can construct a complex message or just use one of our helper functions:
		//   - api.NewCodeBlockMessage("body", true)
		//   - api.NewPlaintextMessage("body", true)
		Message api.Message
		// Messages holds a collection of messages that should be dispatched to the user in the context of a given command execution.
		// To avoid spamming, you can specify max 15 messages.
		// Limitations:
		//   - It's available only for SocketSlack. In the future, it may be adopted across other platforms.
		//   - Message filtering is not available.
		Messages []api.Message
	}
)

// ProtocolVersion is the version that must match between Botkube core
// and Botkube plugins. This should be bumped whenever a change happens in
// one or the other that makes it so that they can't safely communicate.
// This could be adding a new interface value, it could be how helper/schema computes diffs, etc.
//
// NOTE: In the future we can consider using VersionedPlugins. These can be used to negotiate
// a compatible version between client and server. If this is set, Handshake.ProtocolVersion is not required.
const ProtocolVersion = 3

var _ plugin.GRPCPlugin = &Plugin{}

// Plugin This is the implementation of plugin.GRPCPlugin, so we can serve and consume different Botkube Executors.
type Plugin struct {
	// The GRPC plugin must still implement the Plugin interface.
	plugin.NetRPCUnsupportedPlugin

	// Executor represents a concrete implementation that handles the business logic.
	Executor Executor
}

// GRPCServer registers plugin for serving with the given GRPCServer.
func (p *Plugin) GRPCServer(_ *plugin.GRPCBroker, s *grpc.Server) error {
	RegisterExecutorServer(s, &grpcServer{
		Impl: p.Executor,
	})
	return nil
}

// GRPCClient returns the interface implementation for the plugin that is serving via gRPC by GRPCServer.
func (p *Plugin) GRPCClient(_ context.Context, _ *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &grpcClient{
		client: NewExecutorClient(c),
	}, nil
}

type grpcClient struct {
	client ExecutorClient
}

func (p *grpcClient) Execute(ctx context.Context, in ExecuteInput) (ExecuteOutput, error) {
	grpcInput := &ExecuteRequest{
		Command: in.Command,
		Configs: in.Configs,
		Context: &ExecuteContext{
			IsInteractivitySupported: in.Context.IsInteractivitySupported,
			KubeConfig:               in.Context.KubeConfig,
			Message: &MessageContext{
				Text:             in.Context.Message.Text,
				Url:              in.Context.Message.URL,
				ParentActivityId: in.Context.Message.ParentActivityID,
				User: &UserContext{
					Mention:     in.Context.Message.User.Mention,
					DisplayName: in.Context.Message.User.DisplayName,
				},
			},
			IncomingWebhook: &IncomingWebhookContext{
				BaseSourceURL: in.Context.IncomingWebhook.BaseSourceURL,
			},
		},
	}

	if in.Context.IsInteractivitySupported && in.Context.SlackState != nil {
		rawState, err := json.Marshal(in.Context.SlackState)
		if err != nil {
			return ExecuteOutput{}, fmt.Errorf("while marshaling slack state: %w", err)
		}
		grpcInput.Context.SlackState = rawState
	}

	res, err := p.client.Execute(ctx, grpcInput)
	if err != nil {
		return ExecuteOutput{}, err
	}

	extract := func(in []byte) (api.Message, error) {
		if len(in) == 0 || string(in) == "" {
			return api.Message{}, nil
		}

		var msg api.Message
		if err := json.Unmarshal(in, &msg); err != nil {
			return api.Message{}, fmt.Errorf("while unmarshalling message from JSON: %w", err)
		}

		return msg, nil
	}
	msg, err := extract(res.Message)
	if err != nil {
		return ExecuteOutput{}, err
	}

	var msgs []api.Message
	for _, item := range res.Messages[:mathx.Min(maxMessageNumberForSingleCommandExecution, len(res.Messages))] {
		casted, err := extract(item)
		if err != nil {
			return ExecuteOutput{}, err
		}

		msgs = append(msgs, casted)
	}

	return ExecuteOutput{
		Message:  msg,
		Messages: msgs,
	}, nil
}

func (p *grpcClient) Metadata(ctx context.Context) (api.MetadataOutput, error) {
	resp, err := p.client.Metadata(ctx, &emptypb.Empty{})
	if err != nil {
		return api.MetadataOutput{}, err
	}

	return api.MetadataOutput{
		Version:          resp.Version,
		Description:      resp.Description,
		DocumentationURL: resp.DocumentationUrl,
		JSONSchema: api.JSONSchema{
			Value:  resp.GetJsonSchema().GetValue(),
			RefURL: resp.GetJsonSchema().GetRefUrl(),
		},
		Dependencies: api.ConvertDependenciesToAPI(resp.Dependencies),
		Recommended:  resp.Recommended,
	}, nil
}

func (p *grpcClient) Help(ctx context.Context) (api.Message, error) {
	resp, err := p.client.Help(ctx, &emptypb.Empty{})
	if err != nil {
		return api.Message{}, err
	}
	var msg api.Message
	if err := json.Unmarshal(resp.Help, &msg); err != nil {
		return api.Message{}, fmt.Errorf("while unmarshalling help from JSON: %w", err)
	}
	return msg, nil
}

type grpcServer struct {
	UnimplementedExecutorServer
	Impl Executor
}

func (p *grpcServer) Execute(ctx context.Context, request *ExecuteRequest) (*ExecuteResponse, error) {
	var slackState slack.BlockActionStates
	if request.Context != nil && request.Context.SlackState != nil {
		if err := json.Unmarshal(request.Context.SlackState, &slackState); err != nil {
			return nil, fmt.Errorf("while unmarshalling slack state from JSON: %w", err)
		}
	}

	out, err := p.Impl.Execute(ctx, ExecuteInput{
		Command: request.Command,
		Configs: request.Configs,
		Context: ExecuteInputContext{
			SlackState:               &slackState,
			IsInteractivitySupported: request.Context.IsInteractivitySupported,
			KubeConfig:               request.Context.KubeConfig,
			Message:                  p.toMessageIfPresent(request.Context.Message),
			IncomingWebhook: IncomingWebhookDetailsContext{
				BaseSourceURL: request.Context.IncomingWebhook.BaseSourceURL,
			},
		},
	})
	if err != nil {
		return nil, err
	}

	marshalled, err := json.Marshal(out.Message)
	if err != nil {
		return nil, fmt.Errorf("while marshalling help to JSON: %w", err)
	}

	var encodedMsgs [][]byte
	for _, item := range out.Messages {
		marshalled, err := json.Marshal(item)
		if err != nil {
			return nil, fmt.Errorf("while marshalling help to JSON: %w", err)
		}

		encodedMsgs = append(encodedMsgs, marshalled)
	}

	return &ExecuteResponse{
		Message:  marshalled,
		Messages: encodedMsgs,
	}, nil
}

func (*grpcServer) toMessageIfPresent(msg *MessageContext) Message {
	if msg == nil {
		return Message{}
	}
	var user User
	if msg.User != nil {
		user = User{
			Mention:     msg.User.Mention,
			DisplayName: msg.User.DisplayName,
		}
	}

	return Message{
		Text:             msg.Text,
		URL:              msg.Url,
		ParentActivityID: msg.ParentActivityId,
		User:             user,
	}
}

func (p *grpcServer) Metadata(ctx context.Context, _ *emptypb.Empty) (*MetadataResponse, error) {
	meta, err := p.Impl.Metadata(ctx)
	if err != nil {
		return nil, err
	}
	return &MetadataResponse{
		Version:          meta.Version,
		Description:      meta.Description,
		DocumentationUrl: meta.DocumentationURL,
		JsonSchema: &JSONSchema{
			Value:  meta.JSONSchema.Value,
			RefUrl: meta.JSONSchema.RefURL,
		},
		Dependencies: api.ConvertDependenciesFromAPI[*Dependency, Dependency](meta.Dependencies),
		Recommended:  meta.Recommended,
	}, nil
}

func (p *grpcServer) Help(ctx context.Context, _ *emptypb.Empty) (*HelpResponse, error) {
	help, err := p.Impl.Help(ctx)
	if err != nil {
		return nil, err
	}
	marshalled, err := json.Marshal(help)
	if err != nil {
		return nil, fmt.Errorf("while marshalling help to JSON: %w", err)
	}
	return &HelpResponse{
		Help: marshalled,
	}, nil
}

// Serve serves given plugins.
func Serve(p map[string]plugin.Plugin) {
	plugin.Serve(&plugin.ServeConfig{
		Plugins: p,
		HandshakeConfig: plugin.HandshakeConfig{
			ProtocolVersion:  ProtocolVersion,
			MagicCookieKey:   api.HandshakeConfig.MagicCookieKey,
			MagicCookieValue: api.HandshakeConfig.MagicCookieValue,
		},
		GRPCServer: plugin.DefaultGRPCServer,
	})
}
