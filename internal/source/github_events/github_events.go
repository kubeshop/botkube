package github_events

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"time"

	"github.com/google/go-github/v53/github"
	"github.com/google/go-querystring/query"
	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/internal/source/github_events/templates"
	"github.com/kubeshop/botkube/pkg/api/source"
)

const (
	prEventName  = "PullRequestEvent"
	perPageItems = 100
)

// Watcher watches for GitHub events.
type Watcher struct {
	cli             *github.Client
	log             logrus.FieldLogger
	cfg             []RepositoryConfig
	refreshDuration time.Duration
	lastProcessTime map[string]time.Time
	repos           map[string]matchCriteria
	prMatcher       *PullRequestMatcher
	jsonPathMatcher *JSONPathMatcher
}

// NewWatcher returns a new Watcher instance.
func NewWatcher(refreshDuration time.Duration, repositories []RepositoryConfig, cli *github.Client, log logrus.FieldLogger) (*Watcher, error) {
	repos, lastProcessTime, err := normalizeRepos(repositories)
	if err != nil {
		return nil, err
	}

	return &Watcher{
		refreshDuration: refreshDuration,
		cfg:             repositories,
		repos:           repos,
		cli:             cli,
		log:             log,
		prMatcher:       NewPullRequestMatcher(log, cli),
		jsonPathMatcher: NewJSONPathMatcher(log),
		lastProcessTime: lastProcessTime,
	}, nil
}

type matchCriteria struct {
	RepoOwner string
	RepoName  string
	Matchers  On
}

func normalizeRepos(in []RepositoryConfig) (map[string]matchCriteria, map[string]time.Time, error) {
	repos := map[string]matchCriteria{}
	lastProcessTime := map[string]time.Time{}

	for _, repo := range in {
		split := strings.Split(repo.Name, "/")
		if len(split) != 2 {
			return nil, nil, fmt.Errorf(`Wrong repository name. Expected pattern "owner/repository", got %q`, repo.Name)
		}

		existing := repos[repo.Name]
		if repo.OnMatchers.PullRequests != nil && len(repo.OnMatchers.PullRequests) == 0 {
			// to make sure that we also emit events for:
			//   repositories:
			//    - name: owner/repo1
			//      on:
			//        pullRequests: []
			existing.Matchers.PullRequests = append(existing.Matchers.PullRequests, PullRequest{})
		}
		existing.Matchers.PullRequests = append(existing.Matchers.PullRequests, repo.OnMatchers.PullRequests...)

		if len(repo.OnMatchers.EventsAPI) > 0 {
			existing.Matchers.EventsAPI = append(existing.Matchers.EventsAPI, repo.OnMatchers.EventsAPI...)
		}

		repos[repo.Name] = matchCriteria{
			RepoOwner: split[0],
			RepoName:  split[1],
			Matchers:  existing.Matchers,
		}
		lastProcessTime[repo.Name] = time.Now().Add(-repo.BeforeDuration)
	}

	return repos, lastProcessTime, nil
}

func (w *Watcher) AsyncConsumeEvents(ctx context.Context, stream *source.StreamOutput) {
	go func() {
		timer := time.NewTimer(w.refreshDuration)
		defer timer.Stop()

		defer close(stream.Event)
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				w.log.Debug("Checking for new GitHub events in all registered repositories...")
				w.visitAllRepositories(ctx, func(repo matchCriteria, events []CommonEvent) error {
					log := w.log.WithFields(logrus.Fields{
						"repoName":  repo.RepoName,
						"repoOwner": repo.RepoOwner,
					})

					log.WithField("eventsNo", len(events)).Debug("Checking events...")

					w.emitMatchingEvent(ctx, stream, repo, events)
					w.lastProcessTime[repoKey(repo)] = time.Now()
					return nil
				})

				timer.Reset(w.refreshDuration)
			}
		}
	}()
}

func (w *Watcher) emitMatchingEvent(ctx context.Context, stream *source.StreamOutput, repo matchCriteria, events []CommonEvent) {
	for _, ev := range events {
		log := w.log.WithField("gotEvent", ev.Type())

		log.Debug("Checking event")

		messageRenderer := templates.Get(ev.Type())

		switch ev.Type() {
		case prEventName:
			payload, err := ev.ParsePayload()
			if err != nil {
				log.WithError(err).Errorf("while parsing event %q from %s/%s", ev.Type(), repo.RepoOwner, repo.RepoName)
				continue // let's check other events
			}
			var pullRequest *github.PullRequest
			switch pr := payload.(type) {
			case *github.PullRequestEvent:
				pullRequest = pr.PullRequest
			case *github.PullRequest:
				pullRequest = pr
			default:
				w.log.Warnf("Got unknown event type %T", payload)
				continue
			}

			for _, criteria := range repo.Matchers.PullRequests {
				if !w.prMatcher.IsEventMatchingCriteria(ctx, criteria, pullRequest) {
					continue
				}

				msg, err := messageRenderer(ev.GetEvent(), payload, criteria.NotificationTemplate.ToOptions()...)
				if err != nil {
					log.WithError(err).Errorf("while rendering event %q from %s/%s", ev.Type(), repo.RepoOwner, repo.RepoName)
					continue // let's check other events
				}
				stream.Event <- source.Event{
					Message: msg,
				}
			}
		default:
			raw := ev.GetEvent()
			if raw == nil || raw.RawPayload == nil {
				continue
			}

			for _, criteria := range repo.Matchers.EventsAPI {
				if criteria.Type != ev.Type() {
					continue
				}

				payload, err := ev.ParsePayload()
				if err != nil {
					log.WithError(err).Errorf("while parsing event %q from %s/%s", ev.Type(), repo.RepoOwner, repo.RepoName)
					continue // let's check other events
				}

				if !w.jsonPathMatcher.IsEventMatchingCriteria(ev.GetEvent().GetRawPayload(), criteria.JSONPath, criteria.Value) {
					continue
				}

				msg, err := messageRenderer(ev.GetEvent(), payload, criteria.NotificationTemplate.ToOptions()...)
				if err != nil {
					log.WithError(err).Errorf("while rendering event %q from %s/%s", ev.Type(), repo.RepoOwner, repo.RepoName)
					continue // let's check other events
				}
				stream.Event <- source.Event{
					Message: msg,
				}
			}
		}
	}
}

type repositoryEventsProcessor func(repo matchCriteria, events []CommonEvent) error

func (w *Watcher) visitAllRepositories(ctx context.Context, process repositoryEventsProcessor) {
	for name, repo := range w.repos {
		var eventsToProcess []CommonEvent

		repoEvents, err := w.listRepositoryEvents(ctx, repo)
		if err != nil {
			w.log.WithError(err).Error("Failed to list repository events")
		}
		eventsToProcess = append(eventsToProcess, repoEvents...)

		// List pull requests to make sure that we catch all fresh ones
		prs, err := w.listAllPullRequests(ctx, repo)
		if err != nil {
			w.log.WithError(err).Error("Failed to list repository pull requests")
		}
		eventsToProcess = append(eventsToProcess, prs...)

		if len(eventsToProcess) == 0 {
			continue
		}

		err = process(repo, eventsToProcess)
		if err != nil {
			w.log.WithError(err).Errorf("Failed to process events for %s", name)
		}
	}
}

func (w *Watcher) listAllPullRequests(ctx context.Context, repo matchCriteria) ([]CommonEvent, error) {
	w.log.Debugf("Listing all %d last updated pull request", perPageItems)
	pr, r, err := w.List(ctx, repo.RepoOwner, repo.RepoName, &github.PullRequestListOptions{
		State:     "all",
		Sort:      "updated", // we are interested in recent events
		Direction: "desc",
		ListOptions: github.ListOptions{
			PerPage: perPageItems,
		},
	}, w.lastProcessTime[repoKey(repo)])
	if err != nil {
		return nil, fmt.Errorf("while fetching closed pull requests: %w", err)
	}
	if r.StatusCode >= 400 {
		return nil, fmt.Errorf("got unexpected status code: %d", r.StatusCode)
	}

	var eventsToProcess []CommonEvent
	for _, e := range pr {
		if e.UpdatedAt.Before(w.lastProcessTime[repoKey(repo)]) {
			w.log.WithField("prNumber", e.GetNumber()).Debug("Ignoring old pull requests")
			continue
		}
		eventsToProcess = append(eventsToProcess, &GitHubPullRequest{
			RepoName:    fmt.Sprintf("%s/%s", repo.RepoOwner, repo.RepoName),
			PullRequest: e,
		})
	}
	w.log.Debugf("Selected %d PRs to process", len(eventsToProcess))
	return eventsToProcess, nil
}

// List the pull requests for the specified repository.
//
// GitHub API docs: https://docs.github.com/en/rest/pulls/pulls#list-pull-requests
func (w *Watcher) List(ctx context.Context, owner string, repo string, opts *github.PullRequestListOptions, since time.Time) ([]*github.PullRequest, *github.Response, error) {
	u := fmt.Sprintf("repos/%v/%v/pulls", owner, repo)
	u, err := addOptions(u, opts)
	if err != nil {
		return nil, nil, err
	}

	req, err := w.cli.NewRequest("GET", u, nil)
	if err != nil {
		return nil, nil, err
	}

	req.Header.Add("if-modified-since", since.Format(http.TimeFormat))
	var pulls []*github.PullRequest
	resp, err := w.cli.Do(ctx, req, &pulls)
	if err != nil {
		return nil, resp, err
	}

	return pulls, resp, nil
}

// addOptions adds the parameters in opts as URL query parameters to s. opts
// must be a struct whose fields may contain "url" tags.
func addOptions(s string, opts interface{}) (string, error) {
	v := reflect.ValueOf(opts)
	if v.Kind() == reflect.Ptr && v.IsNil() {
		return s, nil
	}

	u, err := url.Parse(s)
	if err != nil {
		return s, err
	}

	qs, err := query.Values(opts)
	if err != nil {
		return s, err
	}

	u.RawQuery = qs.Encode()
	return u.String(), nil
}

// listRepositoryEvents lists all repository events.
// This API is not built to serve real-time use cases. Depending on the time of day, event latency can be anywhere from 30s to 6h.
// source: https://docs.github.com/en/rest/activity/events?apiVersion=2022-11-28#list-repository-events
func (w *Watcher) listRepositoryEvents(ctx context.Context, repo matchCriteria) ([]CommonEvent, error) {
	w.log.Debug("Listing all emitted github events")

	var events []*github.Event
	opts := github.ListOptions{
		PerPage: perPageItems,
	}
	for {
		resultPage, r, err := w.cli.Activity.ListRepositoryEvents(ctx, repo.RepoOwner, repo.RepoName, &opts)
		if err != nil {
			return nil, fmt.Errorf("while listing repository events: %w", err)
		}
		if r.StatusCode >= 400 {
			return nil, fmt.Errorf("got unexpected status code: %d", r.StatusCode)
		}

		events = append(events, resultPage...)
		if r.NextPage == 0 {
			break
		}
		opts.Page = r.NextPage
	}

	var eventsToProcess []CommonEvent
	for _, e := range events {
		if e == nil || e.GetCreatedAt().Before(w.lastProcessTime[repoKey(repo)]) {
			w.log.WithField("eventType", e.GetType()).Debug("Ignoring old events")
			continue
		}
		if e.GetType() == prEventName {
			w.log.Debugf("Ignore %s as we list them on our own", prEventName)
			continue
		}
		w.log.WithFields(logrus.Fields{
			"eventType":      e.GetType(),
			"eventCreatedAt": e.GetCreatedAt().String(),
		}).Debug()
		eventsToProcess = append(eventsToProcess, &GitHubEvent{e})
	}

	w.log.Debugf("Selected %d events to process", len(eventsToProcess))
	return eventsToProcess, nil
}

func repoKey(in matchCriteria) string {
	return fmt.Sprintf("%s/%s", in.RepoOwner, in.RepoName)
}
