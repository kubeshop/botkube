package bot

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/sirupsen/logrus"
	"github.com/sourcegraph/conc/pool"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kubeshop/botkube/internal/config/remote"
	"github.com/kubeshop/botkube/pkg/api/cloudplatform"
	pb "github.com/kubeshop/botkube/pkg/api/cloudteams"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/grpcx"
)

type grpcCloudTeamsConnector struct {
	log          logrus.FieldLogger
	grpcConn     *grpc.ClientConn
	remoteConfig remote.Config

	agentActivityWorkers *pool.Pool
	cloudActivityWorkers *pool.Pool

	activityClient pb.CloudTeams_StreamActivityClient
}

func newGrpcCloudTeamsConnector(log logrus.FieldLogger, cfg config.GRPCServer) (*grpcCloudTeamsConnector, error) {
	remoteConfig, ok := remote.GetConfig()
	if !ok {
		return nil, fmt.Errorf("while getting remote config for %q", config.CloudTeamsCommPlatformIntegration)
	}

	log.WithFields(logrus.Fields{
		"url":                  cfg.URL,
		"disableSecurity":      cfg.DisableTransportSecurity,
		"tlsUseSystemCertPool": cfg.TLS.UseSystemCertPool,
		"tlsCACertificateLen":  len(cfg.TLS.CACertificate),
		"tlsSkipVerify":        cfg.TLS.InsecureSkipVerify,
	}).Debug("Creating gRPC connection to Cloud Teams...")

	creds, err := grpcx.ClientTransportCredentials(log, cfg)
	if err != nil {
		return nil, fmt.Errorf("while creating gRPC credentials: %w", err)
	}

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(creds),
		grpc.WithStreamInterceptor(cloudplatform.AddStreamingClientCredentials(remoteConfig)),
		grpc.WithUnaryInterceptor(cloudplatform.AddUnaryClientCredentials(remoteConfig)),
	}

	conn, err := grpc.Dial(cfg.URL, opts...)
	if err != nil {
		return nil, fmt.Errorf("while creating gRCP connection: %w", err)
	}

	return &grpcCloudTeamsConnector{
		log:          log,
		grpcConn:     conn,
		remoteConfig: remoteConfig,

		cloudActivityWorkers: pool.New().WithMaxGoroutines(platformMessageWorkersCount),
		agentActivityWorkers: pool.New().WithMaxGoroutines(platformMessageWorkersCount),
	}, nil
}

func (c *grpcCloudTeamsConnector) Shutdown() {
	c.log.Info("Shutting down Cloud Teams message processor...")

	if c.activityClient != nil {
		if err := c.activityClient.CloseSend(); err != nil {
			c.log.WithError(err).Error("Cannot closing gRPC stream activity connection")
		}
	}

	if err := c.grpcConn.Close(); err != nil {
		c.log.WithError(err).Error("Cannot close gRPC connection")
	}

	c.cloudActivityWorkers.Wait()
	c.agentActivityWorkers.Wait()
}

func (c *grpcCloudTeamsConnector) Start(ctx context.Context) error {
	activityClient, err := pb.NewCloudTeamsClient(c.grpcConn).StreamActivity(ctx)
	if err != nil {
		return fmt.Errorf("while initializing gRPC cloud client: %w", err)
	}

	c.activityClient = activityClient

	return nil
}

func (c *grpcCloudTeamsConnector) ProcessAgentActivity(ctx context.Context, agentActivity chan *pb.AgentActivity) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case msg, ok := <-agentActivity:
			if !ok {
				return nil
			}
			if msg == nil {
				continue
			}
			c.agentActivityWorkers.Go(func() {
				err := c.activityClient.Send(msg)
				if err != nil {
					c.log.WithError(err).Error("Failed to send Agent activity message")
					return
				}
			})
		}
	}
}

type handleStreamFn func(context.Context, *pb.CloudActivity) (*pb.AgentActivity, error)

func (c *grpcCloudTeamsConnector) ProcessCloudActivity(ctx context.Context, handleCloudActivityFn handleStreamFn) error {
	cloudActivity := make(chan *pb.CloudActivity, platformMessageChannelSize)

	go func() {
		c.log.Info("Starting Cloud Teams message processor...")
		defer c.log.Info("Stopped Cloud Teams message processor...")

		for msg := range cloudActivity {
			if len(msg.Event) == 0 {
				continue
			}
			c.cloudActivityWorkers.Go(func() {
				resp, err := handleCloudActivityFn(ctx, msg)
				if err != nil {
					c.log.WithError(err).Error("Failed to handle Cloud Teams activity")
					return
				}

				if resp == nil {
					return
				}
				err = c.activityClient.Send(resp)
				if err != nil {
					c.log.WithError(err).Error("Failed to send response to Cloud Teams activity")
					return
				}
			})
		}
	}()

	for {
		data, err := c.activityClient.Recv()
		switch err {
		case nil:
		case io.EOF:
			close(cloudActivity)
			c.log.Warn("gRPC connection was closed by server")
			return errors.New("gRPC connection closed")
		default:
			errStatus, ok := status.FromError(err)
			if ok && errStatus.Code() == codes.Canceled && errStatus.Message() == context.Canceled.Error() {
				c.log.Debugf("Context was cancelled...")
				return nil
			}
			return fmt.Errorf("while receiving Cloud Teams events: %w", err)
		}

		select {
		case <-ctx.Done():
			close(cloudActivity)
			c.log.Warn("shutdown requested")
			return ctx.Err()
		case cloudActivity <- data:
		default:
		}
	}
}
