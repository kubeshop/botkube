package format_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/botkube/pkg/events"
	"github.com/kubeshop/botkube/pkg/format"
)

func TestShortMessage(t *testing.T) {
	// given
	testCases := []struct {
		Name     string
		Input    events.Event
		Expected string
	}{
		{},
	}

	for _, tC := range testCases {
		t.Run(tC.Name, func(t *testing.T) {
			// when
			actual := format.ShortMessage(tC.Input)

			// then
			assert.Equal(t, tC.Expected, actual)
		})
	}
}
