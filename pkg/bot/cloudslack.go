package bot

import (
	"context"
	"fmt"

	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	pb "github.com/kubeshop/botkube/pkg/api/cloudslack"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
)

var _ Bot = &CloudSlack{}

// CloudSlack listens for user's message, execute commands and sends back the response.
type CloudSlack struct {
	log logrus.FieldLogger
	cfg config.CloudSlack
}

func NewCloudSlack(log logrus.FieldLogger,
	commGroupName string,
	cfg config.CloudSlack,
	executorFactory ExecutorFactory,
	reporter FatalErrorAnalyticsReporter) (*CloudSlack, error) {
	return &CloudSlack{
		log: log.WithField("integration", config.CloudSlackCommPlatformIntegration),
		cfg: cfg,
	}, nil
}

func (b *CloudSlack) Start(ctx context.Context) error {
	creds := grpc.WithTransportCredentials(insecure.NewCredentials())
	opts := []grpc.DialOption{creds}

	conn, err := grpc.Dial(b.cfg.Server, opts...)
	if err != nil {
		return err
	}
	defer conn.Close()

	pb.NewCloudSlackClient(conn)
	req := &pb.ConnectRequest{
		InstanceId: "123",
	}
	conOpts := []grpc.CallOption{}
	c, err := pb.NewCloudSlackClient(conn).Connect(ctx, conOpts...)
	if err != nil {
		return err
	}

	err = c.Send(req)
	if err != nil {
		return err
	}

	data, err := c.Recv()
	if err != nil {
		return err
	}
	fmt.Printf("received: %q\n", data.Event)

	err = c.CloseSend()
	if err != nil {
		return err
	}
	return nil
}

func (b *CloudSlack) SendMessage(ctx context.Context, msg interactive.CoreMessage, sourceBindings []string) error {
	return nil
}

func (b *CloudSlack) SendMessageToAll(ctx context.Context, msg interactive.CoreMessage) error {
	return nil
}

func (b *CloudSlack) Type() config.IntegrationType {
	return config.BotIntegrationType
}

func (b *CloudSlack) IntegrationName() config.CommPlatformIntegration {
	return config.CloudSlackCommPlatformIntegration
}
