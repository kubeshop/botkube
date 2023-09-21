package analytics

import (
	"runtime"

	"github.com/denisbrodbeck/machineid"
	"github.com/pkg/errors"
	segment "github.com/segmentio/analytics-go"
	"go.szostok.io/version"

	"github.com/kubeshop/botkube/internal/analytics"
	"github.com/kubeshop/botkube/internal/cli"
	"github.com/kubeshop/botkube/internal/loggerx"
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

	isLoggedIn, err := r.reportAndSaveIdentity(id)
	if err != nil {
		return err
	}

	err = r.client.Enqueue(segment.Track{
		AnonymousId: id,
		Event:       cmd,
		Properties:  newProperties(id, isLoggedIn),
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

func (r *SegmentReporter) reportAndSaveIdentity(machineID string) (bool, error) {
	c := &cli.Config{}
	if err := c.Read(); err != nil {
		return false, errors.Wrap(err, "failed to read config")
	}

	isLoggedIn := c.Token != ""
	if c.Identity == machineID {
		return isLoggedIn, nil
	}

	err := r.client.Enqueue(segment.Identify{
		AnonymousId: machineID,
		Traits:      newProperties(machineID, isLoggedIn),
	})
	if err != nil {
		return isLoggedIn, errors.Wrap(err, "failed to report identity")
	}

	c.Identity = machineID
	if err := c.Save(); err != nil {
		return isLoggedIn, errors.Wrap(err, "failed to save config")
	}

	return isLoggedIn, nil
}

func newProperties(id string, cloudLogin bool) map[string]interface{} {
	v := defaultCliVersion
	if vals := version.Get(); vals != nil {
		v = vals.Version
	}
	return map[string]interface{}{
		"OS":          runtime.GOOS,
		"version":     v,
		"machine_id":  id,
		"cloud_login": cloudLogin,
	}
}
