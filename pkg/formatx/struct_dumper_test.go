package formatx_test

import (
	"testing"

	"github.com/MakeNowJust/heredoc"
	"github.com/stretchr/testify/assert"

	"github.com/kubeshop/botkube/pkg/formatx"
)

func TestStructDumper(t *testing.T) {
	type Thread struct {
		TimeStamp int64
		Team      string
	}
	type Message struct {
		Text    string
		UserID  int
		Threads []Thread
	}

	got := formatx.StructDumper().Sdump(Message{
		Text:   "Hello, Botkube!",
		UserID: 3,
		Threads: []Thread{
			{
				TimeStamp: int64(2344442424),
				Team:      "MetalHead",
			},
		},
	})
	expected := heredoc.Doc(`
		formatx_test.Message{
		  Text: "Hello, Botkube!",
		  UserID: 3,
		  Threads: []formatx_test.Thread{
		    formatx_test.Thread{
		      TimeStamp: 2344442424,
		      Team: "MetalHead",
		    },
		  },
		}`)
	assert.Equal(t, expected, got)
}
