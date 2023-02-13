package action

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"html/template"
	"strings"

	sprig "github.com/go-task/slim-sprig"
	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/event"
	"github.com/kubeshop/botkube/pkg/execute"
	"github.com/kubeshop/botkube/pkg/execute/command"
	"github.com/kubeshop/botkube/pkg/multierror"
	"github.com/kubeshop/botkube/pkg/sliceutil"
)

const (
	// unknownValue defines an unknown string value.
	unknownValue = "unknown"
)

// ExecutorFactory facilitates creation of execute.Executor instances.
type ExecutorFactory interface {
	NewDefault(cfg execute.NewDefaultInput) execute.Executor
}

// Provider provides automations for events.
type Provider struct {
	log             logrus.FieldLogger
	cfg             config.Actions
	executorFactory ExecutorFactory
}

// NewProvider returns new instance of Provider.
func NewProvider(log logrus.FieldLogger, cfg config.Actions, executorFactory ExecutorFactory) *Provider {
	return &Provider{log: log, cfg: cfg, executorFactory: executorFactory}
}

// RenderedActionsForEvent finds and processes actions for given event.
func (p *Provider) RenderedActionsForEvent(e event.Event, sourceBindings []string) ([]event.Action, error) {
	var actions []event.Action
	errs := multierror.New()
	for _, action := range p.cfg {
		if !action.Enabled {
			continue
		}

		if !sliceutil.Intersect(sourceBindings, action.Bindings.Sources) {
			continue
		}

		p.log.Debugf("Rendering Action %q (command: %q)...", action.DisplayName, action.Command)
		renderingData := renderingData{
			Event: e,
		}
		renderedCmd, err := p.renderActionCommand(action, renderingData)
		if err != nil {
			errs = multierror.Append(errs, err)
			continue
		}

		p.log.Debugf("Rendered command: %q", renderedCmd)

		actions = append(actions, event.Action{
			DisplayName:      action.DisplayName,
			Command:          fmt.Sprintf("%s %s", api.MessageBotNamePlaceholder, renderedCmd),
			ExecutorBindings: action.Bindings.Executors,
		})
	}

	return actions, errs.ErrorOrNil()
}

// ExecuteEventAction executes action for given event.
func (p *Provider) ExecuteEventAction(ctx context.Context, action event.Action) interactive.CoreMessage {
	e := p.executorFactory.NewDefault(execute.NewDefaultInput{
		Conversation: execute.Conversation{
			IsAuthenticated:  true,
			ExecutorBindings: action.ExecutorBindings,
			CommandOrigin:    command.AutomationOrigin,
			Alias:            unknownValue,
			ID:               unknownValue,
		},
		CommGroupName:   unknownValue,
		Platform:        unknownValue,
		NotifierHandler: &universalNotifierHandler{},
		Message:         strings.TrimSpace(strings.TrimPrefix(action.Command, api.MessageBotNamePlaceholder)),
		User:            fmt.Sprintf("Automation %q", action.DisplayName),
	})
	response := e.Execute(ctx)

	return response
}

type renderingData struct {
	Event event.Event
}

func (p *Provider) renderActionCommand(action config.Action, data renderingData) (string, error) {
	tpl := template.New("action-cmd").Funcs(sprig.FuncMap())
	tpl, err := tpl.Parse(action.Command)
	if err != nil {
		return "", fmt.Errorf("while parsing command template %q for Action %q: %w", action.Command, action.DisplayName, err)
	}

	var result bytes.Buffer
	err = tpl.Execute(&result, data)
	if err != nil {
		return "", fmt.Errorf("while rendering command %q for Action %q: %w", action.Command, action.DisplayName, err)
	}

	return result.String(), nil
}

type universalNotifierHandler struct{}

func (n *universalNotifierHandler) NotificationsEnabled(_ string) bool {
	return false
}

func (n *universalNotifierHandler) SetNotificationsEnabled(_ string, _ bool) error {
	return errors.New("setting notification from automated action is not supported. Use Botkube commands on a specific channel to set notifications")
}
