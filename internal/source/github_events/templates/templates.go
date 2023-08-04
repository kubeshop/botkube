package templates

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v53/github"

	"github.com/kubeshop/botkube/internal/ptr"
	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/formatx"
)

//	Available events:
//	  - CommitCommentEvent
//	  - CreateEvent
//	  - DeleteEvent
//	  - ForkEvent
//	  - GollumEvent
//	  - IssueCommentEvent
//	  - IssuesEvent
//	  - MemberEvent
//	  - PublicEvent
//	  - PullRequestEvent
//	  - PullRequestReviewEvent
//	  - PullRequestReviewCommentEvent
//	  - PullRequestReviewThreadEvent
//	  - PushEvent
//	  - ReleaseEvent
//	  - SponsorshipEvent
//	  - WatchEvent
//
// source: https://docs.github.com/en/webhooks-and-events/events/github-event-types
var templates = map[string]RenderFn{
	"PullRequestEvent": pullRequestEventMessage,

	// WatchEvent for now emitted only when someone stars a repository.
	// https://docs.github.com/en/webhooks-and-events/events/github-event-types#watchevent
	"WatchEvent": watchEventMessage,
}

type RenderFn func(ghEvent *github.Event, event any, opts ...MessageMutatorOption) api.Message

func Get(eventType string) RenderFn {
	fn, found := templates[eventType]
	if !found || fn == nil {
		return genericJSONEventMessage
	}
	return fn
}

type (
	MessageMutatorOption func(message api.Message, payload any) api.Message
)

func pullRequestEventMessage(_ *github.Event, event any, opts ...MessageMutatorOption) api.Message {
	pr, ok := event.(*github.PullRequest)
	if !ok {
		return api.Message{}
	}

	var fields api.TextFields

	fields = append(fields, api.TextField{Key: "Author", Value: pr.GetUser().GetLogin()})
	fields = append(fields, api.TextField{Key: "Merged", Value: strconv.FormatBool(!pr.GetMergedAt().IsZero())})

	var labels []string
	for _, l := range pr.Labels {
		labels = append(labels, ptr.ToValue(l.Name))
	}
	if len(labels) > 0 {
		fields = append(fields, api.TextField{Key: "Labels", Value: fmt.Sprintf("`%s`", strings.Join(labels, "`, `"))})
	}

	fields = append(fields, api.TextField{Key: "Draft", Value: strconv.FormatBool(pr.GetDraft())})

	btnBuilder := api.NewMessageButtonBuilder()
	baseMessage := api.Message{
		Sections: []api.Section{
			{
				Base: api.Base{
					Header:      fmt.Sprintf("Pull request %s", pr.GetState()),
					Description: fmt.Sprintf("#%d %s", pr.GetNumber(), pr.GetTitle()),
				},
				TextFields: fields,
				Buttons: []api.Button{
					btnBuilder.ForURL("Open on GitHub", pr.GetHTMLURL(), api.ButtonStylePrimary),
				},
				Context: []api.ContextItem{
					{
						Text: fmt.Sprintf("Last updated at %s", pr.GetUpdatedAt().Format(time.RFC822)),
					},
				},
			},
		},
	}

	for _, mutator := range opts {
		baseMessage = mutator(baseMessage, pr)
	}
	return baseMessage
}

func genericJSONEventMessage(_ *github.Event, event any, opts ...MessageMutatorOption) api.Message {
	raw, err := json.MarshalIndent(event, "", "  ")
	if err != nil {
		return api.Message{}
	}

	baseMessage := api.Message{
		Sections: []api.Section{
			{
				Base: api.Base{
					Header: "GitHub Event",
					Body: api.Body{
						CodeBlock: string(raw),
					},
				},
			},
		},
	}

	for _, mutator := range opts {
		baseMessage = mutator(baseMessage, event)
	}
	return baseMessage
}

func watchEventMessage(ghEvent *github.Event, event any, opts ...MessageMutatorOption) api.Message {
	var fields api.TextFields
	fields = append(fields, api.TextField{Key: "Repository", Value: ghEvent.GetRepo().GetName()})
	fields = append(fields, api.TextField{Key: "User", Value: formatx.AdaptiveCodeBlock(ghEvent.GetActor().GetLogin())})
	fields = append(fields, api.TextField{Key: "Starred At", Value: ghEvent.GetCreatedAt().Format(time.RFC822)})

	btnBuilder := api.NewMessageButtonBuilder()
	baseMessage := api.Message{
		Sections: []api.Section{
			{
				Base: api.Base{
					Header: "Starred a repository",
				},
				TextFields: fields,
				Buttons: []api.Button{
					btnBuilder.ForURL("View user profile", fmt.Sprintf("https://github.com/%s", ghEvent.GetActor().GetLogin()), api.ButtonStylePrimary),
				},
			},
		},
	}

	for _, mutator := range opts {
		baseMessage = mutator(baseMessage, event)
	}
	return baseMessage
}
