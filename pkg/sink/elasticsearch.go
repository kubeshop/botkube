package sink

import (
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	v4 "github.com/aws/aws-sdk-go/aws/signer/v4"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/olivere/elastic/v7"
	"github.com/sha1sum/aws_signing_client"
	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/multierror"
	"github.com/kubeshop/botkube/pkg/notifier"
	"github.com/kubeshop/botkube/pkg/sliceutil"
)

var _ Sink = &Elasticsearch{}

const (
	// indexSuffixFormat is the date format that would be appended to the index name
	indexSuffixFormat = "2006-01-02" // YYYY-MM-DD
	// awsService for the AWS client to authenticate against
	awsService = "es"
	// AWS Role ARN from POD env variable while using IAM Role for service account
	awsRoleARNEnvName = "AWS_ROLE_ARN"
	// The token file mount path in POD env variable while using IAM Role for service account
	// #nosec G101
	awsWebIDTokenFileEnvName = "AWS_WEB_IDENTITY_TOKEN_FILE"

	elasticErrorReasonResourceAlreadyExists = "resource_already_exists_exception"
)

// Elasticsearch provides integration with the Elasticsearch solution.
type Elasticsearch struct {
	log            logrus.FieldLogger
	reporter       AnalyticsReporter
	client         *elastic.Client
	indices        map[string]config.ELSIndex
	clusterVersion string
	status         notifier.StatusMsg
	failureReason  notifier.FailureReasonMsg
}

// NewElasticsearch creates a new Elasticsearch instance.
func NewElasticsearch(log logrus.FieldLogger, c config.Elasticsearch, reporter AnalyticsReporter) (*Elasticsearch, error) {
	var elsClient *elastic.Client
	var err error

	var elsOpts []elastic.ClientOptionFunc
	switch c.LogLevel {
	case "info":
		elsOpts = append(elsOpts, elastic.SetInfoLog(log))
	case "error":
		elsOpts = append(elsOpts, elastic.SetInfoLog(log), elastic.SetErrorLog(log))
	case "trace":
		elsOpts = append(elsOpts, elastic.SetInfoLog(log), elastic.SetErrorLog(log), elastic.SetTraceLog(log))
	}

	if c.AWSSigning.Enabled {
		// Get credentials from environment variables and create the AWS Signature Version 4 signer
		sess := session.Must(session.NewSession())

		// Use OIDC token to generate credentials if using IAM to Service Account
		awsRoleARN := os.Getenv(awsRoleARNEnvName)
		awsWebIdentityTokenFile := os.Getenv(awsWebIDTokenFileEnvName)
		var creds *credentials.Credentials
		if awsRoleARN != "" && awsWebIdentityTokenFile != "" {
			svc := sts.New(sess)
			p := stscreds.NewWebIdentityRoleProviderWithOptions(svc, awsRoleARN, "", stscreds.FetchTokenPath(awsWebIdentityTokenFile))
			creds = credentials.NewCredentials(p)
		} else if c.AWSSigning.RoleArn != "" {
			creds = stscreds.NewCredentials(sess, c.AWSSigning.RoleArn)
		} else {
			creds = ec2rolecreds.NewCredentials(sess)
		}

		signer := v4.NewSigner(creds)
		awsClient, err := aws_signing_client.New(signer, nil, awsService, c.AWSSigning.AWSRegion)
		if err != nil {
			return nil, fmt.Errorf("while creating new AWS Signing client: %w", err)
		}
		elsOpts = append(elsOpts,
			elastic.SetURL(c.Server),
			elastic.SetScheme("https"),
			elastic.SetHttpClient(awsClient),
			elastic.SetSniff(false),
			elastic.SetHealthcheck(false),
			elastic.SetGzip(false),
		)
	} else {
		elsOpts = append(elsOpts,
			elastic.SetURL(c.Server),
			elastic.SetBasicAuth(c.Username, c.Password),
			elastic.SetSniff(false),
			elastic.SetHealthcheck(false),
			elastic.SetGzip(true),
		)

		if c.SkipTLSVerify {
			tr := &http.Transport{
				// #nosec G402
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
			httpClient := &http.Client{Transport: tr}
			elsOpts = append(elsOpts, elastic.SetHttpClient(httpClient))
		}
	}

	elsClient, err = elastic.NewClient(elsOpts...)
	if err != nil {
		return nil, fmt.Errorf("while creating new Elastic client: %w", err)
	}
	pong, _, err := elsClient.Ping(c.Server).Do(context.Background())
	if err != nil {
		return nil, fmt.Errorf("while pinging cluster: %w", err)
	}

	esNotifier := &Elasticsearch{
		log:            log,
		reporter:       reporter,
		client:         elsClient,
		indices:        c.Indices,
		clusterVersion: pong.Version.Number,
		status:         notifier.StatusUnknown,
		failureReason:  "",
	}

	err = reporter.ReportSinkEnabled(esNotifier.IntegrationName())
	if err != nil {
		return nil, fmt.Errorf("while reporting analytics: %w", err)
	}

	return esNotifier, nil
}

type mapping struct {
	Settings settings `json:"settings"`
}

type settings struct {
	Index index `json:"index"`
}

type index struct {
	Shards   int `json:"number_of_shards"`
	Replicas int `json:"number_of_replicas"`
}

func (e *Elasticsearch) flushIndex(ctx context.Context, indexCfg config.ELSIndex, event interface{}) error {
	// Construct the ELS Index Name with timestamp suffix
	indexName := indexCfg.Name + "-" + time.Now().Format(indexSuffixFormat)
	// Create index if not exists
	exists, err := e.client.IndexExists(indexName).Do(ctx)
	if err != nil {
		return fmt.Errorf("while getting index: %w", err)
	}
	if !exists {
		// Create a new index.
		mapping := mapping{
			Settings: settings{
				index{
					Shards:   indexCfg.Shards,
					Replicas: indexCfg.Replicas,
				},
			},
		}
		_, err := e.client.CreateIndex(indexName).BodyJson(mapping).Do(ctx)
		if err != nil && elastic.ErrorReason(err) != elasticErrorReasonResourceAlreadyExists {
			return fmt.Errorf("while creating index: %w", err)
		}
	}

	// Send event to els
	indexService := e.client.Index().Index(indexName)
	majorVersion, err := esMajorClusterVersion(e.clusterVersion)
	if err != nil {
		return fmt.Errorf("while getting cluster major version: %w", err)
	}
	if majorVersion <= 7 && indexCfg.Type != "" {
		// Only Elasticsearch <= 7.x supports Type parameter
		// nolint:staticcheck
		indexService.Type(indexCfg.Type)
	}
	_, err = indexService.BodyJson(event).Do(ctx)
	if err != nil {
		return fmt.Errorf("while posting data to ELS: %w", err)
	}
	_, err = e.client.Flush().Index(indexName).Do(ctx)
	if err != nil {
		return fmt.Errorf("while flushing data in ELS: %w", err)
	}
	e.log.Debugf("Event successfully sent to Elasticsearch index %s", indexName)
	return nil
}

// SendEvent sends an event to a configured elasticsearch server.
func (e *Elasticsearch) SendEvent(ctx context.Context, rawData any, sources []string) error {
	e.log.Debugf(">> Sending to Elasticsearch: %+v", rawData)

	errs := multierror.New()
	for _, indexCfg := range e.indices {
		if !sliceutil.Intersect(indexCfg.Bindings.Sources, sources) {
			continue
		}
		err := e.flushIndex(ctx, indexCfg, rawData)
		if err != nil {
			e.setFailureReason(notifier.FailureReasonConnectionError)
			errs = multierror.Append(errs, fmt.Errorf("while sending event to Elasticsearch index %q: %w", indexCfg.Name, err))
			continue
		}

		e.setFailureReason("")
		e.log.Debugf("Event successfully sent to Elasticsearch index %q", indexCfg.Name)
	}

	return errs.ErrorOrNil()
}

// IntegrationName describes the notifier integration name.
func (e *Elasticsearch) IntegrationName() config.CommPlatformIntegration {
	return config.ElasticsearchCommPlatformIntegration
}

// Type describes the notifier type.
func (e *Elasticsearch) Type() config.IntegrationType {
	return config.SinkIntegrationType
}

func (e *Elasticsearch) setFailureReason(reason notifier.FailureReasonMsg) {
	if reason == "" {
		e.status = notifier.StatusHealthy
	} else {
		e.status = notifier.StatusUnHealthy
	}
	e.failureReason = reason
}

// GetStatus gets sink status
func (e *Elasticsearch) GetStatus() notifier.Status {
	return notifier.Status{
		Status:   e.status,
		Restarts: "0/0",
		Reason:   e.failureReason,
	}
}

func esMajorClusterVersion(v string) (int, error) {
	versionParts := strings.Split(v, ".")
	if len(versionParts) == 1 {
		return 0, errors.New("cluster version is not valid")
	}
	majorVersion, err := strconv.Atoi(versionParts[0])
	if err != nil {
		return 0, fmt.Errorf("failed to parse cluster version: %s", versionParts[0])
	}
	return majorVersion, nil
}
