package argocd

import (
	"fmt"

	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/config"
)

type IncomingRequestContext struct {
	App           *config.K8sResourceRef `json:"app"`
	DetailsUIPath *string                `json:"detailsUiPath"`
	RepoURL       *string                `json:"repoUrl"`
}
type IncomingRequestBody struct {
	Message api.Message            `json:"message"`
	Context IncomingRequestContext `json:"context"`
}

func (s *Source) generateInteractivitySection(reqBody IncomingRequestBody, cfg Config) *api.Section {
	var section api.Section
	if reqBody.Context.App != nil && len(cfg.Interactivity.CommandVerbs) > 0 {
		var opts []api.OptionItem
		for _, verb := range cfg.Interactivity.CommandVerbs {
			opts = append(opts, api.OptionItem{
				Name:  verb,
				Value: fmt.Sprintf("%s application %s --namespace %s", verb, reqBody.Context.App.Name, reqBody.Context.App.Namespace),
			})
		}
		cmdPrefix := fmt.Sprintf("%s kubectl", api.MessageBotNamePlaceholder)
		section.Selects = api.Selects{
			ID: "",
			Items: []api.Select{
				{
					Name:    "Run command...",
					Command: cmdPrefix,
					OptionGroups: []api.OptionGroup{
						{
							Name:    "Supported commands",
							Options: opts,
						},
					},
				},
			},
		}
	}

	btnBldr := api.NewMessageButtonBuilder()
	if cfg.Interactivity.EnableViewInUIButton && s.shouldDisplayUIDetails(reqBody, cfg) {
		section.Buttons = append(section.Buttons, btnBldr.ForURL("View in UI", fmt.Sprintf("%s%s", cfg.ArgoCD.UIBaseURL, *reqBody.Context.DetailsUIPath)))
	}

	if cfg.Interactivity.EnableOpenRepositoryButton && s.shouldDisplayOpenRepo(reqBody) {
		section.Buttons = append(section.Buttons, btnBldr.ForURL("Open repository", *reqBody.Context.RepoURL))
	}

	if len(section.Buttons) == 0 && len(section.Selects.Items) == 0 {
		return nil
	}

	return &section
}

func (s *Source) generateNonInteractiveFields(reqBody IncomingRequestBody, cfg Config) []api.TextField {
	var out []api.TextField
	if s.shouldDisplayOpenRepo(reqBody) {
		out = append(out, api.TextField{
			Key:   "Repository",
			Value: *reqBody.Context.RepoURL,
		})
	}

	if s.shouldDisplayUIDetails(reqBody, cfg) {
		out = append(out, api.TextField{
			Key:   "Link to ArgoCD UI",
			Value: fmt.Sprintf("%s%s", cfg.ArgoCD.UIBaseURL, *reqBody.Context.DetailsUIPath),
		})
	}

	return out
}

func (s *Source) shouldDisplayUIDetails(reqBody IncomingRequestBody, cfg Config) bool {
	return cfg.ArgoCD.UIBaseURL != "" && reqBody.Context.DetailsUIPath != nil && *reqBody.Context.DetailsUIPath != ""
}

func (s *Source) shouldDisplayOpenRepo(reqBody IncomingRequestBody) bool {
	return reqBody.Context.RepoURL != nil && *reqBody.Context.RepoURL != ""
}
