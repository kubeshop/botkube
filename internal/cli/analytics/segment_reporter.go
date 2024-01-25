package analytics

import (
	"github.com/kubeshop/botkube/pkg/loggerx"
	"runtime"

	"github.com/denisbrodbeck/machineid"
	"github.com/pkg/errors"
	segment "github.com/segmentio/analytics-go"
	"go.szostok.io/version"

	"github.com/kubeshop/botkube/internal/analytics"
	"github.com/kubeshop/botkube/internal/cli"
)

var _ Reporter = &SegmentReporter{}

// SegmentReporter is a Segment implementation of the Reporter interface.
type SegmentReporter struct {
	client segment.Client
}

// NewSegmentReporter creates a new SegmentReporter instance.
func NewSegmentReporter(apiKey string) (Reporter, error) {
	c, err := segment.NewWithConfig(apiKey, segment.Config{
		Logger:  analytics.NewSegmentLoggerAdapter(loggerx.NewNoop()),
		Verbose: false,
	})
	return &SegmentReporter{client: c}, err
}

// ReportCommand reports a command to the analytics service.
func (r *SegmentReporter) ReportCommand(cmd string) error {
	id, err := machineid.ID()
	if err != nil {
		return errors.Wrap(err, "failed to get machine identity")
	}
	c := cli.NewConfig()
	isLoggedIn := c.IsUserLoggedIn()
	properties := newProperties(id, isLoggedIn)

	err = r.client.Enqueue(segment.Identify{
		AnonymousId: id,
		Traits:      properties,
	})

	if err != nil {
		return errors.Wrap(err, "failed to report identity")
	}

	err = r.client.Enqueue(segment.Track{
		AnonymousId: id,
		Event:       cmd,
		Properties:  properties,
	})

	if err != nil {
		return errors.Wrap(err, "failed to report command")
	}

	return nil
}

// ReportError reports an error to the analytics service.
func (r *SegmentReporter) ReportError(in error, cmd string) error {
	id, err := machineid.ID()
	if err != nil {
		return errors.Wrap(err, "failed to get machine identity")
	}
	err = r.client.Enqueue(segment.Track{
		AnonymousId: id,
		Event:       "error",
		Properties: segment.NewProperties().
			Set("reason", in.Error()).
			Set("command", cmd),
	})
	if err != nil {
		return errors.Wrap(err, "failed to report error")
	}
	return nil
}

// Close closes the reporter.
func (r *SegmentReporter) Close() {
	_ = r.client.Close()
}

func newProperties(id string, cloudLogin bool) map[string]interface{} {
	v := defaultCliVersion
	if vals := version.Get(); vals != nil {
		v = vals.Version
	}
	return map[string]interface{}{
		"OS":              runtime.GOOS,
		"arch":            runtime.GOARCH,
		"version":         v,
		"machine_id":      id,
		"cloud_logged_in": cloudLogin,
	}
}
