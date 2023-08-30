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

func (s *Source) generateInteractivitySection(reqBody IncomingRequestBody) *api.Section {
	var section api.Section
	if reqBody.Context.App != nil && len(s.cfg.Interactivity.CommandVerbs) > 0 {
		var opts []api.OptionItem
		for _, verb := range s.cfg.Interactivity.CommandVerbs {
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
	if s.cfg.ArgoCD.UIBaseURL != "" && reqBody.Context.DetailsUIPath != nil && *reqBody.Context.DetailsUIPath != "" {
		section.Buttons = append(section.Buttons, btnBldr.ForURL("Details", fmt.Sprintf("%s%s", s.cfg.ArgoCD.UIBaseURL, *reqBody.Context.DetailsUIPath), api.ButtonStylePrimary))
	}

	if reqBody.Context.RepoURL != nil && *reqBody.Context.RepoURL != "" {
		section.Buttons = append(section.Buttons, btnBldr.ForURL("Repository", *reqBody.Context.RepoURL))
	}

	if len(section.Buttons) == 0 && len(section.Selects.Items) == 0 {
		return nil
	}

	return &section
}
