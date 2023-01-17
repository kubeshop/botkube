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
	// Api refers to prometheus client. https://github.com/prometheus/client_golang
	Api    promApi.API
	alerts sync.Map
}

type GetAlertsRequest struct {
	IgnoreOldAlerts bool
	MinAlertTime    time.Time
	AlertStates     []promApi.AlertState
}

type alert promApi.Alert

func (a *alert) IsOld(request GetAlertsRequest) bool {
	return request.IgnoreOldAlerts && a.ActiveAt.Before(request.MinAlertTime)
}

func (a *alert) HasValidState(request GetAlertsRequest) bool {
	for _, state := range request.AlertStates {
		if a.State == state {
			return true
		}
	}
	return false
}

// NewClient initializes prometheus client
func NewClient(url string) *Client {
	c, _ := promClient.NewClient(promClient.Config{
		Address: url,
	})

	newAPI := promApi.NewAPI(c)
	return &Client{
		Api: newAPI,
	}
}

// Alerts returns only new alerts.
func (c *Client) Alerts(ctx context.Context, request GetAlertsRequest) ([]alert, error) {
	alerts, err := c.Api.Alerts(ctx)
	if err != nil {
		return nil, err
	}
	var newAlerts []alert
	for _, al := range alerts.Alerts {
		a := alert(al)
		if a.IsOld(request) || !a.HasValidState(request) {
			continue
		}
		key := fmt.Sprintf("%+v", a.Labels)
		if value, ok := c.alerts.Load(key); !ok || a.State != value.(promApi.Alert).State {
			newAlerts = append(newAlerts, a)
			c.alerts.Store(key, a)
		}
	}
	return newAlerts, nil
}
