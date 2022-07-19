package analytics_test

import (
	"fmt"
	"strconv"
	"time"

	segment "github.com/segmentio/analytics-go"
)

var _ segment.Client = &fakeSegmentCli{}

type fakeSegmentCli struct {
	messages []segment.Message
}

func (f *fakeSegmentCli) Close() error {
	return nil
}

func (f *fakeSegmentCli) Enqueue(message segment.Message) error {
	err := message.Validate()
	if err != nil {
		return err
	}

	msg, err := f.setInternalFields(message)
	if err != nil {
		return err
	}

	f.messages = append(f.messages, msg)
	return nil
}

func (f *fakeSegmentCli) setInternalFields(message segment.Message) (segment.Message, error) {
	timestamp := time.Date(2009, 11, 17, 20, 34, 58, 651387237, time.UTC)
	id := strconv.Itoa(len(f.messages))

	switch m := message.(type) {
	case segment.Group:
		m.Type = "group"
		m.MessageId = id
		m.Timestamp = timestamp
		return m, nil
	case segment.Identify:
		m.Type = "identify"
		m.MessageId = id
		m.Timestamp = timestamp
		return m, nil
	case segment.Track:
		m.Type = "track"
		m.MessageId = id
		m.Timestamp = timestamp
		return m, nil

	default:
		return nil, fmt.Errorf("unknown type %T", message)
	}
}
