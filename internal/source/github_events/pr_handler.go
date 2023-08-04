package github_events

import (
	"context"

	"github.com/google/go-github/v53/github"
	"github.com/sirupsen/logrus"
)

// PullRequestMatcher knows how to validate if a given GitHub pull request matches defines criteria.
type PullRequestMatcher struct {
	log logrus.FieldLogger
	cli *github.Client
}

// NewPullRequestMatcher returns a new PullRequestMatcher instance.
func NewPullRequestMatcher(log logrus.FieldLogger, cli *github.Client) *PullRequestMatcher {
	return &PullRequestMatcher{
		log: log,
		cli: cli,
	}
}

// IsEventMatchingCriteria returns true if PR matches all defined criteria.
func (w *PullRequestMatcher) IsEventMatchingCriteria(ctx context.Context, criteria PullRequest, pullRequest *github.PullRequest) bool {
	repo := pullRequest.GetBase().GetRepo()
	name, owner := repo.GetName(), repo.GetOwner().GetLogin()

	log := w.log.WithFields(logrus.Fields{
		"owner":    owner,
		"name":     name,
		"prNumber": pullRequest.GetNumber(),
	})
	if !w.hasRequiredState(criteria.Types, pullRequest) {
		log.Debug("PR doesn't have required state")
		return false
	}

	if !w.hasRequiredLabels(criteria.Labels, pullRequest) {
		log.Debug("PR doesn't have required labels")
		return false
	}

	if !w.isChangingRequiredFiles(ctx, criteria.Paths, pullRequest) {
		log.Debug("PR doesn't change required files")
		return false
	}

	log.Debug("Pull request matched all required criteria")
	return true
}

func (w *PullRequestMatcher) hasRequiredLabels(labels IncludeExcludeRegex, pullRequest *github.PullRequest) bool {
	if labels.IsEmpty() {
		return true
	}

	for _, label := range pullRequest.Labels {
		defined, err := labels.IsDefined(label.GetName())
		if err != nil {
			w.log.WithError(err).WithFields(logrus.Fields{
				"gotLabel": label.GetName(),
				"prNumber": pullRequest.GetNumber(),
			}).Errorf("while checking required labels")
			continue
		}
		if defined {
			return true
		}
	}

	return false
}

func (w *PullRequestMatcher) isChangingRequiredFiles(ctx context.Context, paths IncludeExcludeRegex, pullRequest *github.PullRequest) bool {
	if paths.IsEmpty() {
		return true
	}

	// consider using GitHub GraphQL API:
	// query PullRequestByNumber($owner: String!, $repo: String!, $pr_number: Int!) {
	//		repository(owner: $owner, name: $repo) {
	//			pullRequest(number: $pr_number) {%s}
	//		}
	//	}

	repo := pullRequest.GetBase().GetRepo()
	name, owner := repo.GetName(), repo.GetOwner().GetLogin()

	files, _, err := w.cli.PullRequests.ListFiles(ctx, owner, name, pullRequest.GetNumber(), &github.ListOptions{})
	if err != nil {
		w.log.WithError(err).WithFields(logrus.Fields{
			"owner":    owner,
			"name":     name,
			"prNumber": pullRequest.GetNumber(),
		}).Errorf("while listing pull request files")
		return false
	}

	for _, f := range files {
		fileName := f.GetPreviousFilename() // previous as it might be changed on PR
		if fileName == "" {
			fileName = f.GetFilename()
		}
		defined, err := paths.IsDefined(fileName)
		if err != nil {
			w.log.WithError(err).WithFields(logrus.Fields{
				"fileName": f.Filename,
				"name":     f.PreviousFilename,
				"prNumber": pullRequest.GetNumber(),
			}).Errorf("while checking required files")
			continue
		}
		if defined {
			return true
		}
	}
	return false
}

func (w *PullRequestMatcher) hasRequiredState(types []string, pullRequest *github.PullRequest) bool {
	if len(types) == 0 {
		return true
	}

	for _, prType := range types {
		switch prType {
		case "merged":
			if pullRequest.GetState() != "closed" || !pullRequest.GetMerged() {
				w.log.Debug("PR is not merged")
				continue
			}
			return true
		default:
			if pullRequest.GetState() != prType {
				w.log.WithFields(logrus.Fields{
					"gotState": pullRequest.GetState(),
					"expState": prType,
				}).Debug("PR is not in state")
				continue
			}
			return true
		}
	}
	return false
}
