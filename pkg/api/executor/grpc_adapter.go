package executor

import (
	"context"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/kubeshop/botkube/pkg/api"
)

// Executor defines the Botkube executor plugin functionality.
type Executor interface {
	Execute(context.Context, ExecuteInput) (ExecuteOutput, error)
	Metadata(ctx context.Context) (api.MetadataOutput, error)
}

type (
	// ExecuteInput holds the input of the Execute function.
	ExecuteInput struct {
		// Command holds the command to be executed.
		Command string
		// Configs is a list of Executor configurations specified by users.
		Configs []*Config
	}

	// ExecuteOutput holds the output of the Execute function.
	ExecuteOutput struct {
		// Data represent the output of processing a given input command.
		Data string
	}
)

// ProtocolVersion is the version that must match between Botkube core
// and Botkube plugins. This should be bumped whenever a change happens in
// one or the other that makes it so that they can't safely communicate.
// This could be adding a new interface value, it could be how helper/schema computes diffs, etc.
//
// NOTE: In the future we can consider using VersionedPlugins. These can be used to negotiate
// a compatible version between client and server. If this is set, Handshake.ProtocolVersion is not required.
const ProtocolVersion = 1

var _ plugin.GRPCPlugin = &Plugin{}

// Plugin This is the implementation of plugin.GRPCPlugin, so we can serve and consume different Botkube Executors.
type Plugin struct {
	// The GRPC plugin must still implement the Plugin interface.
	plugin.NetRPCUnsupportedPlugin

	// Executor represent a concrete implementation that handles the business logic.
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
	res, err := p.client.Execute(ctx, &ExecuteRequest{
		Command: in.Command,
		Configs: in.Configs,
	})
	if err != nil {
		return ExecuteOutput{}, err
	}
	return ExecuteOutput{
		Data: res.Data,
	}, nil
}

func (p *grpcClient) Metadata(ctx context.Context) (api.MetadataOutput, error) {
	resp, err := p.client.Metadata(ctx, &emptypb.Empty{})
	if err != nil {
		return api.MetadataOutput{}, err
	}

	return api.MetadataOutput{
		Version:     resp.Version,
		Description: resp.Description,
	}, nil
}

type grpcServer struct {
	UnimplementedExecutorServer
	Impl Executor
}

func (p *grpcServer) Execute(ctx context.Context, request *ExecuteRequest) (*ExecuteResponse, error) {
	out, err := p.Impl.Execute(ctx, ExecuteInput{
		Command: request.Command,
		Configs: request.Configs,
	})
	if err != nil {
		return nil, err
	}
	return &ExecuteResponse{
		Data: out.Data,
	}, nil
}

func (p *grpcServer) Metadata(ctx context.Context, _ *emptypb.Empty) (*MetadataResponse, error) {
	out, err := p.Impl.Metadata(ctx)
	if err != nil {
		return nil, err
	}
	return &MetadataResponse{
		Version:     out.Version,
		Description: out.Description,
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
