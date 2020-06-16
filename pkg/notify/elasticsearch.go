// Copyright (c) 2019 InfraCloud Technologies
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package notify

import (
	"context"
	"fmt"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/events"
	"github.com/infracloudio/botkube/pkg/log"
	"github.com/olivere/elastic"
)

// ElasticSearch contains auth cred and index setting
type ElasticSearch struct {
	ELSClient   *elastic.Client
	Server      string
	Index       string
	Shards      int
	Replicas    int
	Type        string
	ClusterName string
}

// NewElasticSearch returns new ElasticSearch object
func NewElasticSearch(c *config.Config) (Notifier, error) {
	// create elasticsearch client
	elsClient, err := elastic.NewClient(elastic.SetURL(c.Communications.ElasticSearch.Server), elastic.SetBasicAuth(c.Communications.ElasticSearch.Username, c.Communications.ElasticSearch.Password), elastic.SetSniff(false), elastic.SetHealthcheck(false), elastic.SetGzip(true))
	if err != nil {
		return nil, err
	}
	return &ElasticSearch{
		ELSClient:   elsClient,
		Index:       c.Communications.ElasticSearch.Index.Name,
		Type:        c.Communications.ElasticSearch.Index.Type,
		Shards:      c.Communications.ElasticSearch.Index.Shards,
		Replicas:    c.Communications.ElasticSearch.Index.Replicas,
		ClusterName: c.Settings.ClusterName,
	}, nil
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

// SendEvent sends event notification to slack
func (e *ElasticSearch) SendEvent(event events.Event) (err error) {
	log.Debug(fmt.Sprintf(">> Sending to ElasticSearch: %+v", event))
	ctx := context.Background()

	// set missing cluster name to event object
	event.Cluster = e.ClusterName

	// Create index if not exists
	exists, err := e.ELSClient.IndexExists(e.Index).Do(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("Failed to get index. Error:%s", err.Error()))
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
		_, err := e.ELSClient.CreateIndex(e.Index).BodyJson(mapping).Do(ctx)
		if err != nil {
			log.Error(fmt.Sprintf("Failed to create index. Error:%s", err.Error()))
			return err
		}
	}

	// Send event to els
	_, err = e.ELSClient.Index().Index(e.Index).Type(e.Type).BodyJson(event).Do(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("Failed to post data to els. Error:%s", err.Error()))
		return err
	}
	_, err = e.ELSClient.Flush().Index(e.Index).Do(ctx)
	if err != nil {
		log.Error(fmt.Sprintf("Failed to flush data to els. Error:%s", err.Error()))
		return err
	}
	log.Debugf("Event successfully sent to ElasticSearch index %s", e.Index)
	return nil
}

// SendMessage sends message to slack channel
func (e *ElasticSearch) SendMessage(msg string) error {
	return nil
}
