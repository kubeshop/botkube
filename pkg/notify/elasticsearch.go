package notify

import (
	"context"
	"fmt"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/events"
	log "github.com/infracloudio/botkube/pkg/logging"
	"github.com/olivere/elastic"
)

// elsClient ElasticSearch client
var elsClient *elastic.Client

// ElasticSearch contains auth cred and index setting
type ElasticSearch struct {
	Username    string
	Password    string
	Server      string
	Index       string
	Shards      int
	Replicas    int
	Type        string
	ClusterName string
}

// NewElasticSearch returns new Slack object
func NewElasticSearch(c *config.Config) Notifier {
	return &ElasticSearch{
		Username:    c.Communications.ElasticSearch.Username,
		Password:    c.Communications.ElasticSearch.Password,
		Server:      c.Communications.ElasticSearch.Server,
		Index:       c.Communications.ElasticSearch.Index.Name,
		Type:        c.Communications.ElasticSearch.Index.Type,
		Shards:      c.Communications.ElasticSearch.Index.Shards,
		Replicas:    c.Communications.ElasticSearch.Index.Replicas,
		ClusterName: c.Settings.ClusterName,
	}
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

func init() {
	c, err := config.New()
	if err != nil {
		log.Logger.Fatal(fmt.Sprintf("Error in loading configuration. Error:%s", err.Error()))
	}
	if !c.Communications.ElasticSearch.Enabled {
		return
	}
	// create elasticsearch client
	elsClient, err = elastic.NewClient(elastic.SetURL(c.Communications.ElasticSearch.Server), elastic.SetBasicAuth(c.Communications.ElasticSearch.Username, c.Communications.ElasticSearch.Password), elastic.SetSniff(false), elastic.SetHealthcheck(false), elastic.SetGzip(true))
	if err != nil {
		log.Logger.Error(fmt.Sprintf("Failed to create els client. Error:%s", err.Error()))
	}
}

// SendEvent sends event notification to slack
func (e *ElasticSearch) SendEvent(event events.Event) (err error) {
	log.Logger.Debug(fmt.Sprintf(">> Sending to ElasticSearch: %+v", event))
	ctx := context.Background()

	// set missing cluster name to event object
	event.Cluster = e.ClusterName

	// Create elsClient if not created
	if elsClient == nil {
		elsClient, err = elastic.NewClient(elastic.SetURL(e.Server), elastic.SetBasicAuth(e.Username, e.Password), elastic.SetSniff(false), elastic.SetHealthcheck(false), elastic.SetGzip(true))
		if err != nil {
			log.Logger.Error(fmt.Sprintf("Failed to create els client. Error:%s", err.Error()))
			return err
		}
	}

	// Create index if not exists
	exists, err := elsClient.IndexExists(e.Index).Do(ctx)
	if err != nil {
		log.Logger.Error(fmt.Sprintf("Failed to get index. Error:%s", err.Error()))
		return err
	}
	if !exists {
		// Create a new index.
		mapping := mapping{
			Settings: settings{
				index{
					Shards:   e.Shards,
					Replicas: e.Replicas,
				},
			},
		}
		_, err := elsClient.CreateIndex(e.Index).BodyJson(mapping).Do(ctx)
		if err != nil {
			log.Logger.Error(fmt.Sprintf("Failed to create index. Error:%s", err.Error()))
			return err
		}
	}

	// Send event to els
	_, err = elsClient.Index().Index(e.Index).Type(e.Type).BodyJson(event).Do(ctx)
	if err != nil {
		log.Logger.Error(fmt.Sprintf("Failed to post data to els. Error:%s", err.Error()))
		return err
	}
	_, err = elsClient.Flush().Index(e.Index).Do(ctx)
	if err != nil {
		log.Logger.Error(fmt.Sprintf("Failed to flush data to els. Error:%s", err.Error()))
		return err
	}
	log.Logger.Debugf("Event successfully sent to ElasticSearch index %s", e.Index)
	return nil
}

// SendMessage sends message to slack channel
func (e *ElasticSearch) SendMessage(msg string) error {
	return nil
}
