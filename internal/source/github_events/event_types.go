package github_events

import (
	"github.com/google/go-github/v53/github"

	"github.com/kubeshop/botkube/pkg/ptr"
)

// CommonEvent defines unified event. As a result we can process both events from /events API and custom e.g. list of all active pull requests.
type CommonEvent interface {
	ParsePayload() (payload any, err error)
	Type() string
	GetEvent() *github.Event
}

type GitHubEvent struct {
	*github.Event
}

func (g *GitHubEvent) Type() string {
	if g == nil {
		return ""
	}
	return ptr.ToValue(g.Event.Type)
}

func (g *GitHubEvent) GetEvent() *github.Event {
	return g.Event
}

type GitHubPullRequest struct {
	RepoName string
	*github.PullRequest
}

func (g *GitHubPullRequest) Type() string {
	return prEventName
}

func (g *GitHubPullRequest) GetEvent() *github.Event {
	return &github.Event{
		Type:  ptr.FromType(g.Type()),
		Actor: g.User,
		Repo: &github.Repository{
			Name: ptr.FromType(g.RepoName),
		},
		CreatedAt: g.CreatedAt,
	}
}

func (g *GitHubPullRequest) ParsePayload() (any, error) {
	return g.PullRequest, nil
}
