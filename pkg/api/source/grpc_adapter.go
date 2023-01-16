package source

import (
	"context"
	"io"
	"log"

	"github.com/hashicorp/go-plugin"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/kubeshop/botkube/pkg/api"
)

// Source defines the Botkube source plugin functionality.
type Source interface {
	Stream(context.Context, StreamInput) (StreamOutput, error)
	Metadata(context.Context) (api.MetadataOutput, error)
}

type (
	// StreamInput holds the input of the Stream function.
	StreamInput struct {
		// Configs is a list of Source configurations specified by users.
		Configs []*Config
	}

	// StreamOutput holds the output of the Stream function.
	StreamOutput struct {
		// Output represents the streamed events. It is from start of plugin execution.
		Output chan []byte
		// TODO: we should consider adding error feedback channel too.
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

// Plugin This is the implementation of plugin.GRPCPlugin, so we can serve and consume different Botkube Sources.
type Plugin struct {
	// The GRPC plugin must still implement the Plugin interface.
	plugin.NetRPCUnsupportedPlugin

	// Source represent a concrete implementation that handles the business logic.
	Source Source
}

// GRPCServer registers plugin for serving with the given GRPCServer.
func (p *Plugin) GRPCServer(_ *plugin.GRPCBroker, s *grpc.Server) error {
	RegisterSourceServer(s, &grpcServer{
		Source: p.Source,
	})
	return nil
}

// GRPCClient returns the interface implementation for the plugin that is serving via gRPC by GRPCServer.
func (p *Plugin) GRPCClient(_ context.Context, _ *plugin.GRPCBroker, c *grpc.ClientConn) (interface{}, error) {
	return &grpcClient{
		client: NewSourceClient(c),
	}, nil
}

type grpcClient struct {
	client SourceClient
}

func (p *grpcClient) Stream(ctx context.Context, in StreamInput) (StreamOutput, error) {
	stream, err := p.client.Stream(ctx, &StreamRequest{
		Configs: in.Configs,
	})
	if err != nil {
		return StreamOutput{}, err
	}

	out := StreamOutput{
		Output: make(chan []byte),
	}

	go func() {
		for {
			// RecvMsg blocks until it receives a message into m or the stream is
			// done. It returns io.EOF when the stream completes successfully.
			feature, err := stream.Recv()
			if err == io.EOF {
				break
			}

			// On any other error, the stream is aborted and the error contains the RPC
			// status.
			if err != nil {
				log.Print(err)
				// TODO: we should consider adding error feedback channel to StreamOutput.
				return
			}
			out.Output <- feature.Output
		}
	}()

	return out, nil
}

func (p *grpcClient) Metadata(ctx context.Context) (api.MetadataOutput, error) {
	resp, err := p.client.Metadata(ctx, &emptypb.Empty{})
	if err != nil {
		return api.MetadataOutput{}, err
	}

	return api.MetadataOutput{
		Version:     resp.Version,
		Description: resp.Description,
		JSONSchema: api.JSONSchema{
			Value:  resp.GetJsonSchema().GetValue(),
			RefURL: resp.GetJsonSchema().GetRefURL(),
		},
	}, nil
}

type grpcServer struct {
	UnimplementedSourceServer
	Source Source
}

func (p *grpcServer) Metadata(ctx context.Context, _ *emptypb.Empty) (*MetadataResponse, error) {
	meta, err := p.Source.Metadata(ctx)
	if err != nil {
		return nil, err
	}
	return &MetadataResponse{
		Version:     meta.Version,
		Description: meta.Description,
		JsonSchema: &JSONSchema{
			Value:  meta.JSONSchema.Value,
			RefURL: meta.JSONSchema.RefURL,
		},
	}, nil
}

func (p *grpcServer) Stream(req *StreamRequest, gstream Source_StreamServer) error {
	ctx := gstream.Context()

	// It's up to the 'Stream' method to close the returned channels as it sends the data to it.
	// We can only use 'ctx' to cancel streaming and release associated resources.
	stream, err := p.Source.Stream(ctx, StreamInput{
		Configs: req.Configs,
	})
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done(): // client canceled stream, we can release this connection.
			return ctx.Err()
		case out, ok := <-stream.Output:
			if !ok {
				return nil // output closed, no more chunk logs
			}

			err := gstream.Send(&StreamResponse{
				Output: out,
			})
			if err != nil {
				return err
			}
		}
	}
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
