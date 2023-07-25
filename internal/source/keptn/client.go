package keptn

import (
	"context"
	"fmt"
	"time"

	api "github.com/keptn/go-utils/pkg/api/utils/v2"
)

// Client Keptn client
type Client struct {
	// API refers to Keptn client. https://github.com/keptn/go-utils
	API *api.APISet
}

// GetEventsRequest represents a request to get events from Keptn.
type GetEventsRequest struct {
	Project  string
	Service  string
	FromTime time.Time
}

// Event represents a Keptn event returned from Keptn API.
type Event struct {
	ID     string
	Source string
	Type   string
	Data   Data
}

// Data represents a Keptn event data which is used by plugin internally.
type Data struct {
	Message string
	Project string
	Service string
	Status  string
	Stage   string
	Result  string
}

// ToAnonymizedEventDetails returns a map of event details which is used for telemetry purposes.
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
	filter := api.EventFilter{
		FromTime: fromTime,
	}
	if request.Project != "" {
		filter.Project = request.Project
	}
	if request.Service != "" {
		filter.Service = request.Service
	}
	res, err := c.API.Events().GetEvents(ctx, &filter, api.EventsGetEventsOptions{})
	if err != nil {
		return nil, err.ToError()
	}

	for _, ev := range res {
		data := Data{}
		err := ev.DataAs(&data)
		if err != nil {
			return nil, fmt.Errorf("while mapping Keptn event to internal event %w", err)
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
