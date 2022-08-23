package sink

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/aws/aws-sdk-go/aws/credentials"
	"github.com/aws/aws-sdk-go/aws/credentials/ec2rolecreds"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	v4 "github.com/aws/aws-sdk-go/aws/signer/v4"
	"github.com/aws/aws-sdk-go/service/sts"
	"github.com/olivere/elastic"
	"github.com/sha1sum/aws_signing_client"
	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/events"
	"github.com/kubeshop/botkube/pkg/multierror"
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
)

// Elasticsearch provides integration with the Elasticsearch solution.
type Elasticsearch struct {
	log      logrus.FieldLogger
	reporter AnalyticsReporter
	client   *elastic.Client
	indices  map[string]config.ELSIndex
}

// NewElasticsearch creates a new Elasticsearch instance.
func NewElasticsearch(log logrus.FieldLogger, c config.Elasticsearch, reporter AnalyticsReporter) (*Elasticsearch, error) {
	var elsClient *elastic.Client
	var err error
	var creds *credentials.Credentials
	if c.AWSSigning.Enabled {
		// Get credentials from environment variables and create the AWS Signature Version 4 signer
		sess := session.Must(session.NewSession())

		// Use OIDC token to generate credentials if using IAM to Service Account
		awsRoleARN := os.Getenv(awsRoleARNEnvName)
		awsWebIdentityTokenFile := os.Getenv(awsWebIDTokenFileEnvName)
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
		elsClient, err = elastic.NewClient(
			elastic.SetURL(c.Server),
			elastic.SetScheme("https"),
			elastic.SetHttpClient(awsClient),
			elastic.SetSniff(false),
			elastic.SetHealthcheck(false),
			elastic.SetGzip(false),
		)
		if err != nil {
			return nil, fmt.Errorf("while creating new Elastic client: %w", err)
		}
	} else {
		elsClientParams := []elastic.ClientOptionFunc{
			elastic.SetURL(c.Server),
			elastic.SetBasicAuth(c.Username, c.Password),
			elastic.SetSniff(false),
			elastic.SetHealthcheck(false),
			elastic.SetGzip(true),
		}

		if c.SkipTLSVerify {
			tr := &http.Transport{
				// #nosec G402
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
			}
			httpClient := &http.Client{Transport: tr}
			elsClientParams = append(elsClientParams, elastic.SetHttpClient(httpClient))
		}
		// create elasticsearch client
		elsClient, err = elastic.NewClient(elsClientParams...)
		if err != nil {
			return nil, fmt.Errorf("while creating new Elastic client: %w", err)
		}
	}

	esNotifier := &Elasticsearch{
		log:      log,
		reporter: reporter,
		client:   elsClient,
		indices:  c.Indices,
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
		if err != nil {
			return fmt.Errorf("while creating index: %w", err)
		}
	}

	// Send event to els
	_, err = e.client.Index().Index(indexName).Type(indexCfg.Type).BodyJson(event).Do(ctx)
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

// SendEvent sends event notification to Elasticsearch
func (e *Elasticsearch) SendEvent(ctx context.Context, event events.Event, eventSources []string) (err error) {
	e.log.Debugf(">> Sending to Elasticsearch: %+v", event)

	errs := multierror.New()
	for _, indexCfg := range e.indices {
		if !sliceutil.Intersect(indexCfg.Bindings.Sources, eventSources) {
			continue
		}

		err := e.flushIndex(ctx, indexCfg, event)
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("while sending event to Elasticsearch index %q: %w", indexCfg.Name, err))
			continue
		}

		e.log.Debugf("Event successfully sent to Elasticsearch index %q", indexCfg.Name)
	}

	return errs.ErrorOrNil()
}

// SendMessage is no-op
func (e *Elasticsearch) SendMessage(_ context.Context, _ string) error {
	return nil
}

// IntegrationName describes the notifier integration name.
func (e *Elasticsearch) IntegrationName() config.CommPlatformIntegration {
	return config.ElasticsearchCommPlatformIntegration
}

// Type describes the notifier type.
func (e *Elasticsearch) Type() config.IntegrationType {
	return config.SinkIntegrationType
}
