package cloudplatform

import (
	"context"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"

	"github.com/kubeshop/botkube/internal/config/remote"
)

const (
	APIKeyContextKey       = "X-Api-Key"       // #nosec
	DeploymentIDContextKey = "X-Deployment-Id" // #nosec
)

func AddStreamingClientCredentials(remoteCfg remote.Config) grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		md := metadata.New(map[string]string{
			APIKeyContextKey:       remoteCfg.APIKey,
			DeploymentIDContextKey: remoteCfg.Identifier,
		})

		ctx = metadata.NewOutgoingContext(ctx, md)

		clientStream, err := streamer(ctx, desc, cc, method, opts...)
		if err != nil {
			return nil, err
		}

		return clientStream, nil
	}
}

func AddUnaryClientCredentials(remoteCfg remote.Config) grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		md := metadata.New(map[string]string{
			APIKeyContextKey:       remoteCfg.APIKey,
			DeploymentIDContextKey: remoteCfg.Identifier,
		})

		ctx = metadata.NewOutgoingContext(ctx, md)
		return invoker(ctx, method, req, reply, cc, opts...)
	}
}
