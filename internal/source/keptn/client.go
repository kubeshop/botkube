package keptn

import (
	"context"
	"time"

	api "github.com/keptn/go-utils/pkg/api/utils/v2"
)

// Client Keptn client
type Client struct {
	// API refers to Keptn client. https://github.com/keptn/go-utils
	API *api.APISet
}

type GetEventsRequest struct {
	Project  string
	FromTime time.Time
}

type Event struct {
	ID     string
	Source string
	Type   string
	Data   Data
}

type Data struct {
	Message string
	Project string
	Service string
	Status  string
	Stage   string
	Result  string
}

func (e *Event) ToAnonymizedEventDetails() map[string]interface{} {
	return map[string]interface{}{
		"ID":     e.ID,
		"Source": e.Source,
		"Type":   e.Type,
	}
}

// NewClient initializes Keptn client
func NewClient(url, token string) (*Client, error) {
	client, err := api.New(url, api.WithAuthToken(token))

	if err != nil {
		return nil, err
	}

	return &Client{
		API: client,
	}, nil
}

// Events returns only new events.
func (c *Client) Events(ctx context.Context, request *GetEventsRequest) ([]Event, error) {
	fromTime := request.FromTime.UTC().Format(time.RFC3339)
	var events []Event
	res, err := c.API.Events().GetEvents(ctx, &api.EventFilter{
		Project:  request.Project,
		FromTime: fromTime,
	}, api.EventsGetEventsOptions{})
	if err != nil {
		return nil, err.ToError()
	}

	for _, ev := range res {
		data := Data{}
		err := ev.DataAs(&data)
		if err != nil {
			return nil, err
		}
		events = append(events, Event{
			ID:     ev.ID,
			Source: *ev.Source,
			Type:   *ev.Type,
			Data:   data,
		})
	}
	return events, nil
}
