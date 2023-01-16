package prometheus

import (
	"context"
	"fmt"
	"time"

	promClient "github.com/prometheus/client_golang/api"
	promApi "github.com/prometheus/client_golang/api/prometheus/v1"
)

const (
	alertStoreTtlInSeconds = 120
)

// Client prometheus client
type Client struct {
	// Api refers to prometheus client. https://github.com/prometheus/client_golang
	Api    promApi.API
	alerts *AlertCache
}

// NewClient initializes prometheus client
func NewClient(url string) *Client {
	c, _ := promClient.NewClient(promClient.Config{
		Address: url,
	})

	newAPI := promApi.NewAPI(c)
	return &Client{
		Api:    newAPI,
		alerts: NewAlertCache(AlertCacheConfig{TTL: alertStoreTtlInSeconds}),
	}
}

// Alerts returns only new alerts.
func (c *Client) Alerts(ctx context.Context) ([]promApi.Alert, error) {
	alerts, err := c.Api.Alerts(ctx)
	if err != nil {
		return nil, err
	}
	var newAlerts []promApi.Alert
	for _, alert := range alerts.Alerts {
		now := time.Now()
		alertTime := alert.ActiveAt
		if alertTime.Add(alertStoreTtlInSeconds * time.Second).After(now) {
			key := fmt.Sprintf("%+v", alert)
			a := c.alerts.Get(key)
			if a.Value == "" {
				newAlerts = append(newAlerts, alert)
				c.alerts.Put(key, alert)
			}
		}
	}
	return newAlerts, nil
}
