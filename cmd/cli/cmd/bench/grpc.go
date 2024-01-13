package bench

import (
	"context"
	"errors"
	"fmt"
	"sync/atomic"

	"github.com/sirupsen/logrus"
	"github.com/sourcegraph/conc/pool"
	"github.com/spf13/cobra"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"
	"golang.org/x/exp/maps"
	"google.golang.org/grpc"

	intconfig "github.com/kubeshop/botkube/internal/config"
	"github.com/kubeshop/botkube/internal/config/remote"
	"github.com/kubeshop/botkube/internal/loggerx"
	"github.com/kubeshop/botkube/pkg/api/cloudplatform"
	pb "github.com/kubeshop/botkube/pkg/api/cloudteams"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/grpcx"
)

type Options struct {
	Requests    int
	Concurrency int
}

func NewGRPC() *cobra.Command {
	var opts Options
	cmd := &cobra.Command{
		Use: "grpc [OPTIONS]",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()

			remoteConfig, ok := remote.GetConfig()
			if !ok {
				return errors.New("failed to get remote config")
			}

			cfg, err := getConfig(ctx, remoteConfig)
			if err != nil {
				return err
			}
			log := loggerx.New(config.Logger{})

			activityClient, err := getClient(ctx, log, cfg, remoteConfig)
			if err != nil {
				return err
			}
			workers := pool.New().WithMaxGoroutines(opts.Concurrency)

			var failed atomic.Int32
			fmt.Printf("Running %d requests against %s gRPC\n", opts.Requests, cfg.URL)

			p := mpb.New(mpb.WithWidth(64))
			bar := p.New(int64(opts.Requests),
				mpb.BarStyle().Lbound("╢").Filler("▌").Tip("▌").Padding("░").Rbound("╟"),
				mpb.PrependDecorators(
					decor.OnComplete(decor.AverageETA(decor.ET_STYLE_GO), "done"),
				),
				mpb.AppendDecorators(decor.Percentage()),
			)

			for i := 0; i < opts.Requests; i++ {
				ii := i
				workers.Go(func() {

					defer bar.Increment()
					err := activityClient.Send(&pb.AgentActivity{
						Message: &pb.Message{
							MessageType: pb.MessageType_MESSAGE_SOURCE,
							Data:        []byte("Load testing"),
						},
					})
					if err != nil {
						failed.Add(1)
						log.Errorf("Could not send %d message: %s", ii, err.Error())
					}
				})
			}

			workers.Wait()
			p.Wait()

			return nil
		},
	}

	flags := cmd.Flags()
	flags.IntVar(&opts.Requests, "n", 100, "Number of requests to run")
	flags.IntVar(&opts.Concurrency, "c", 10, "Number of requests to run concurrently. Total number of requests cannot be smaller than the concurency level.")
	return cmd
}

func getConfig(ctx context.Context, remoteConfig remote.Config) (config.GRPCServer, error) {
	gqlClient := remote.NewDefaultGqlClient(remoteConfig)
	deployClient := remote.NewDeploymentClient(gqlClient)

	cfgProvider := intconfig.GetProvider(true, deployClient)
	configs, _, err := cfgProvider.Configs(ctx)
	if err != nil {
		return config.GRPCServer{}, fmt.Errorf("while loading configuration files: %w", err)
	}
	conf, _, err := config.LoadWithDefaults(configs)
	if err != nil {
		return config.GRPCServer{}, fmt.Errorf("while merging app configuration: %w", err)
	}
	if conf == nil {
		return config.GRPCServer{}, fmt.Errorf("configuration cannot be nil")
	}

	c := maps.Values[map[string]config.Communications](conf.Communications)
	return c[0].CloudTeams.Server, nil
}

func getClient(ctx context.Context, log logrus.FieldLogger, cfg config.GRPCServer, remoteConfig remote.Config) (pb.CloudTeams_StreamActivityClient, error) {
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

	activityClient, err := pb.NewCloudTeamsClient(conn).StreamActivity(ctx)
	if err != nil {
		return nil, fmt.Errorf("while initializing gRPC cloud client: %w", err)
	}

	return activityClient, nil
}
