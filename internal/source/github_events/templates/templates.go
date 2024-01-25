package templates

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/go-github/v53/github"

	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/formatx"
	"github.com/kubeshop/botkube/pkg/ptr"
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

type RenderFn func(ghEvent *github.Event, event any, opts ...MessageMutatorOption) (api.Message, error)

func Get(eventType string) RenderFn {
	fn, found := templates[eventType]
	if !found || fn == nil {
		return genericJSONEventMessage
	}
	return fn
}

type (
	MessageMutatorOption func(message api.Message, payload any) (api.Message, error)
)

func pullRequestEventMessage(gh *github.Event, event any, opts ...MessageMutatorOption) (api.Message, error) {
	pr, ok := event.(*github.PullRequest)
	if !ok {
		return api.Message{}, fmt.Errorf("got unknown event type %T", event)
	}

	var fields api.TextFields

	fields = append(fields, api.TextField{Key: "Repository", Value: gh.GetRepo().GetName()})
	fields = append(fields, api.TextField{Key: "Author", Value: pr.GetUser().GetLogin()})
	fields = append(fields, api.TextField{Key: "State", Value: pr.GetState()})
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
					Header:      "Pull request event",
					Description: fmt.Sprintf("#%d %s", pr.GetNumber(), pr.GetTitle()),
				},
				TextFields: fields,
				Buttons: []api.Button{
					btnBuilder.ForURL("View", pr.GetHTMLURL(), api.ButtonStylePrimary),
				},
				Context: []api.ContextItem{
					{
						Text: fmt.Sprintf("Last updated at %s", pr.GetUpdatedAt().Format(time.RFC822)),
					},
				},
			},
		},
	}

	var err error
	for _, mutator := range opts {
		baseMessage, err = mutator(baseMessage, pr)
		if err != nil {
			return api.Message{}, err
		}
	}
	return baseMessage, nil
}

func genericJSONEventMessage(ghEvent *github.Event, event any, opts ...MessageMutatorOption) (api.Message, error) {
	raw, err := json.MarshalIndent(event, "", "  ")
	if err != nil {
		return api.Message{}, err
	}

	baseMessage := api.Message{
		Sections: []api.Section{
			{
				Base: api.Base{
					Header: fmt.Sprintf("%s GitHub Event", ghEvent.GetType()),
					Body: api.Body{
						CodeBlock: string(raw),
					},
				},
				Context: []api.ContextItem{
					{
						Text: fmt.Sprintf("Created at %s", ghEvent.GetCreatedAt().Format(time.RFC822)),
					},
				},
			},
		},
	}

	for _, mutator := range opts {
		baseMessage, err = mutator(baseMessage, event)
		if err != nil {
			return api.Message{}, err
		}
	}
	return baseMessage, nil
}

func watchEventMessage(ghEvent *github.Event, event any, opts ...MessageMutatorOption) (api.Message, error) {
	var fields api.TextFields
	fields = append(fields, api.TextField{Key: "Repository", Value: ghEvent.GetRepo().GetName()})
	fields = append(fields, api.TextField{Key: "User", Value: formatx.AdaptiveCodeBlock(ghEvent.GetActor().GetLogin())})
	fields = append(fields, api.TextField{Key: "Starred At", Value: ghEvent.GetCreatedAt().Format(time.RFC822)})

	btnBuilder := api.NewMessageButtonBuilder()
	baseMessage := api.Message{
		Sections: []api.Section{
			{
				Base: api.Base{
					Header: "⭐ Starred a repository",
				},
				TextFields: fields,
				Buttons: []api.Button{
					btnBuilder.ForURL("View user profile", fmt.Sprintf("https://github.com/%s", ghEvent.GetActor().GetLogin()), api.ButtonStylePrimary),
				},
			},
		},
	}

	var err error
	for _, mutator := range opts {
		baseMessage, err = mutator(baseMessage, event)
		if err != nil {
			return api.Message{}, err
		}
	}
	return baseMessage, nil
}
