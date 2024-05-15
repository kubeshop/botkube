// Package sink. This file contains a hack functions to extract metadata from different source events to be used in
// PagerDuty payload.
package sink

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/prometheus/common/model"

	k8sconfig "github.com/kubeshop/botkube/internal/source/kubernetes/config"
	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/config"
)

var mdEmojiTag = regexp.MustCompile(`:(\w+):`)

type (
	eventPayload struct {
		k8sEventPayload        `mapstructure:",squash"`
		argoPayload            `mapstructure:",squash"`
		prometheusEventPayload `mapstructure:",squash"`
	}

	k8sEventPayload struct {
		Level     k8sconfig.Level
		Type      string
		Kind      string
		Name      string
		Namespace string
		Messages  []string
	}

	argoPayload struct {
		Message                api.Message
		IncomingRequestContext struct {
			App           *config.K8sResourceRef
			DetailsUIPath *string
			RepoURL       *string
		}
	}

	prometheusEventPayload struct {
		Annotations model.LabelSet
		Labels      model.LabelSet
	}
)

type eventMetadata struct {
	// A brief text summary of the event, used to generate the summaries/titles of any associated alerts.
	// The maximum permitted length of this property is 1024 characters.
	Summary string
	// Component of the source machine that is responsible for the event.
	// source: https://developer.pagerduty.com/api-reference/368ae3d938c9e-send-an-event-to-pager-duty
	Component string
	IsAlert   bool
	Links     []EventLink
}

func enrichWithK8sEventMetadata(out eventMetadata, in k8sEventPayload) eventMetadata {
	if in.Level == k8sconfig.Error {
		out.IsAlert = true
	} else {
		out.IsAlert = false
	}

	if in.Kind != "" && in.Name != "" && in.Namespace != "" {
		out.Component = fmt.Sprintf("%s/%s/%s", in.Kind, in.Namespace, in.Name)
	}

	if len(in.Messages) > 0 {
		out.Summary = strings.Join(in.Messages, "\n")
	}

	if in.Type != "" {
		out.Summary = fmt.Sprintf("[%s] %s", in.Type, out.Summary)
	}

	return out
}

func enrichWithArgoCDEventMetadata(out eventMetadata, in argoPayload) eventMetadata {
	header := in.Message.Sections[0].Header
	header = mdEmojiTag.ReplaceAllString(header, "") // remove all emoji tags

	if header != "" {
		out.Summary = header
	}

	var (
		isDegraded = strings.Contains(out.Summary, "has degraded")
		isFailed   = strings.Contains(out.Summary, "failed")
	)
	if isDegraded || isFailed {
		out.IsAlert = true
	} else {
		out.IsAlert = false
	}

	if in.IncomingRequestContext.RepoURL != nil {
		out.Links = append(out.Links, EventLink{
			Text: "Repository",
			Href: *in.IncomingRequestContext.RepoURL,
		})
	}

	if in.IncomingRequestContext.DetailsUIPath != nil {
		out.Links = append(out.Links, EventLink{
			Text: "Details",
			Href: *in.IncomingRequestContext.DetailsUIPath,
		})
	}

	if in.IncomingRequestContext.App != nil {
		out.Component = fmt.Sprintf("%s/%s", in.IncomingRequestContext.App.Namespace, in.IncomingRequestContext.App.Name)
	}

	return out
}

func enrichWithPrometheusEventMetadata(out eventMetadata, in prometheusEventPayload) eventMetadata {
	out.IsAlert = true // all prometheus events we treat as alerts

	var (
		alertName   = in.Labels["alertname"]
		description = in.Annotations["description"]
	)

	if alertName != "" {
		out.Component = string(alertName)
	}

	if description != "" {
		out.Summary = string(description)
	}

	return out
}
