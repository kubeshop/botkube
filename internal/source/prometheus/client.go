package prometheus

import (
	"context"
	"fmt"
	"sync"
	"time"

	promClient "github.com/prometheus/client_golang/api"
	promApi "github.com/prometheus/client_golang/api/prometheus/v1"
)

// Client prometheus client
type Client struct {
	// API refers to prometheus client. https://github.com/prometheus/client_golang
	API    promApi.API
	alerts sync.Map
}

type GetAlertsRequest struct {
	IgnoreOldAlerts bool
	MinAlertTime    time.Time
	AlertStates     []promApi.AlertState
}

type alert promApi.Alert

// IsValid validates alert
func (a *alert) IsValid(request GetAlertsRequest) bool {
	// Ignore alerts based on their ages only if `IgnoreOldAlerts` configuration is set to true.
	if request.IgnoreOldAlerts && a.ActiveAt.Before(request.MinAlertTime) {
		return false
	}

	// Alert state should be in allowed alert state list.
	for _, state := range request.AlertStates {
		if a.State == state {
			return true
		}
	}
	return false
}

// NewClient initializes Prometheus client
func NewClient(url string) (*Client, error) {
	c, err := promClient.NewClient(promClient.Config{
		Address: url,
	})

	if err != nil {
		return nil, err
	}

	return &Client{
		API: promApi.NewAPI(c),
	}, nil
}

// Alerts returns only new alerts.
func (c *Client) Alerts(ctx context.Context, request GetAlertsRequest) ([]alert, error) {
	alerts, err := c.API.Alerts(ctx)
	if err != nil {
		return nil, err
	}
	var newAlerts []alert
	for _, al := range alerts.Alerts {
		a := alert(al)
		if !a.IsValid(request) {
			continue
		}
		key := fmt.Sprintf("%+v", a.Labels)
		if value, ok := c.alerts.Load(key); !ok || a.State != value.(alert).State {
			newAlerts = append(newAlerts, a)
			c.alerts.Store(key, a)
		}
	}
	return newAlerts, nil
}
