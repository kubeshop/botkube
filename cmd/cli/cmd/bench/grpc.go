package bench

import (
	"context"
	"errors"
	"fmt"
	"github.com/kubeshop/botkube/internal/cli/printer"
	"github.com/morikuni/aec"
	"sync/atomic"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/sourcegraph/conc/pool"
	"github.com/spf13/cobra"
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
	Requests       int
	Concurrency    int
	Burst          int
	BurstSleepTime time.Duration
}

func NewGRPC() *cobra.Command {
	var opts Options
	cmd := &cobra.Command{
		Use: "grpc [OPTIONS]",
		RunE: func(cmd *cobra.Command, args []string) (err error) {
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

			activityClient, err := getClient(ctx, log, cfg.Server, remoteConfig)
			if err != nil {
				return err
			}
			workers := pool.New().WithMaxGoroutines(opts.Concurrency)

			var failed atomic.Int32

			status := printer.NewStatus(cmd.OutOrStdout(), fmt.Sprintf("Running %d requests against %s gRPC\n", opts.Requests, cfg.Server.URL))
			defer func() {
				status.End(err == nil)
				fmt.Println(aec.Show)
			}()

			err = status.InfoStructFields("Run details:", opts)

			dataFmt := `{"baseBody":{"codeBlock":"Load testing: %d"}}`

			for i := 1; i <= opts.Requests; i++ {
				ii := i
				if i%opts.Burst == 0 {
					status.Step("[%d/%d] Pause for %v for next burst to run...", ii, opts.Requests, opts.BurstSleepTime)
					time.Sleep(opts.BurstSleepTime)
				}
				workers.Go(func() {
					err := activityClient.Send(&pb.AgentActivity{
						Message: &pb.Message{
							TeamId:         cfg.Teams[0].ID,
							ConversationId: maps.Values(cfg.Teams[0].Channels)[0].Identifier(),
							MessageType:    pb.MessageType_MESSAGE_SOURCE,
							Data:           []byte(fmt.Sprintf(dataFmt, ii)),
						},
					})
					if err != nil {
						failed.Add(1)
						log.Errorf("Could not send %d message: %s", ii, err.Error())
					}
				})
			}

			workers.Wait()

			fmt.Printf("Total requests: %d\n", opts.Requests)
			fmt.Printf("Failed requests: %d\n", failed.Load())
			return nil
		},
	}

	flags := cmd.Flags()

	flags.IntVar(&opts.Requests, "requests", 1000, "Number of requests to run.")
	flags.IntVar(&opts.Concurrency, "concurrency", 100, "Number of workers")
	flags.IntVar(&opts.Burst, "burst", 30, "Number of requests to send concurrently. Total number shouldn't be higher than the concurrency level.")
	flags.DurationVar(&opts.BurstSleepTime, "burst-sleep-time", 5*time.Second, "Duration to sleep between each burst.")

	return cmd
}

func getConfig(ctx context.Context, remoteConfig remote.Config) (config.CloudTeams, error) {
	gqlClient := remote.NewDefaultGqlClient(remoteConfig)
	deployClient := remote.NewDeploymentClient(gqlClient)

	cfgProvider := intconfig.GetProvider(true, deployClient)
	configs, _, err := cfgProvider.Configs(ctx)
	if err != nil {
		return config.CloudTeams{}, fmt.Errorf("while loading configuration files: %w", err)
	}
	conf, _, err := config.LoadWithDefaults(configs)
	if err != nil {
		return config.CloudTeams{}, fmt.Errorf("while merging app configuration: %w", err)
	}
	if conf == nil {
		return config.CloudTeams{}, fmt.Errorf("configuration cannot be nil")
	}

	c := maps.Values[map[string]config.Communications](conf.Communications)
	return c[0].CloudTeams, nil
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
