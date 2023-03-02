package source

import (
	"context"
	"encoding/json"
	"fmt"
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
		// Context holds streaming context.
		Context StreamInputContext
	}

	// StreamInputContext holds streaming context.
	StreamInputContext struct {
		// IsInteractivitySupported is set to true only if communication platform supports interactive Messages.
		IsInteractivitySupported bool

		// KubeConfig is the path to kubectl configuration file.
		KubeConfig string

		// ClusterName is the name of underlying Kubernetes cluster which is provided by end user.
		ClusterName string
	}

	// StreamOutput holds the output of the Stream function.
	StreamOutput struct {
		// Output represents the streamed events. It is from start of plugin execution.
		// Deprecated: Use the Message field instead.
		//
		// Migration path:
		//
		//	Old approach:
		//	  return executor.StreamOutput{
		//	  	Output: out,
		//	  }
		//
		//	New approach:
		//	  return executor.StreamOutput{
		//	  	Message: api.NewPlaintextMessage(out, true),
		//	  }
		Output chan []byte
		// Event represents the streamed events with message,raw object, and analytics data. It is from start of plugin consumption.
		// You can construct a complex message.data or just use one of our helper functions:
		//   - api.NewCodeBlockMessage("body", true)
		//   - api.NewPlaintextMessage("body", true)
		Event chan Event
	}

	Event struct {
		Message         api.Message
		RawObject       any
		AnalyticsLabels map[string]interface{}
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
	request := &StreamRequest{
		Configs: in.Configs,
		Context: &StreamContext{
			IsInteractivitySupported: in.Context.IsInteractivitySupported,
			KubeConfig:               in.Context.KubeConfig,
			ClusterName:              in.Context.ClusterName,
		},
	}
	stream, err := p.client.Stream(ctx, request)
	if err != nil {
		return StreamOutput{}, err
	}

	out := StreamOutput{
		Output: make(chan []byte),
		Event:  make(chan Event),
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
			var event Event
			if len(feature.Event) != 0 && string(feature.Event) != "" {
				if err := json.Unmarshal(feature.Event, &event); err != nil {
					log.Print(fmt.Errorf("while unmarshalling message from JSON: %w", err))
					return
				}
			}
			out.Output <- feature.Output
			out.Event <- event
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
			RefURL: resp.GetJsonSchema().GetRefUrl(),
		},
		Dependencies: api.ConvertDependenciesToAPI(resp.Dependencies),
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
			RefUrl: meta.JSONSchema.RefURL,
		},
		Dependencies: api.ConvertDependenciesFromAPI[*Dependency, Dependency](meta.Dependencies),
	}, nil
}

func (p *grpcServer) Stream(req *StreamRequest, gstream Source_StreamServer) error {
	ctx := gstream.Context()

	// It's up to the 'Stream' method to close the returned channels as it sends the data to it.
	// We can only use 'ctx' to cancel streaming and release associated resources.
	stream, err := p.Source.Stream(ctx, StreamInput{
		Configs: req.Configs,
		Context: StreamInputContext{
			IsInteractivitySupported: req.Context.IsInteractivitySupported,
			KubeConfig:               req.Context.KubeConfig,
			ClusterName:              req.Context.ClusterName,
		},
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
				return nil // output closed, no more messages
			}

			err := gstream.Send(&StreamResponse{
				Output: out,
			})
			if err != nil {
				return err
			}
		case msg, ok := <-stream.Event:
			if !ok {
				return nil // output closed, no more chunk logs
			}

			marshalled, err := json.Marshal(msg)
			if err != nil {
				return fmt.Errorf("while marshalling msg to byte: %w", err)
			}

			err = gstream.Send(&StreamResponse{
				Event: marshalled,
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
