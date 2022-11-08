package action

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/events"
	"github.com/kubeshop/botkube/pkg/execute"
	"github.com/kubeshop/botkube/pkg/execute/command"
)

const (
	// universalBotNamePlaceholder is a cross-platform placeholder for bot name in commands.
	universalBotNamePlaceholder = "{{BotName}}"

	// unknownValue defines an unknown string value.
	unknownValue = "unknown"
)

// ExecutorFactory facilitates creation of execute.Executor instances.
type ExecutorFactory interface {
	NewDefault(cfg execute.NewDefaultInput) execute.Executor
}

// Provider provides automations for events.
type Provider struct {
	cfg             config.Actions
	executorFactory ExecutorFactory
}

// NewProvider returns new instance of Provider.
func NewProvider(cfg config.Actions, executorFactory ExecutorFactory) *Provider {
	return &Provider{cfg: cfg, executorFactory: executorFactory}
}

// RenderedActionsForEvent finds and processes actions for given event.
// TODO: Implement this as a part of https://github.com/kubeshop/botkube/issues/831
func (p *Provider) RenderedActionsForEvent(event events.Event) ([]events.Action, error) {
	// 1. see if there are any actions for this event configured in config
	// 2. filter out all actions that are not enabled and not applicable for this event
	// 3. for each applicable action, render final command based on the event data

	var actions []events.Action
	//cmd := "kubectl get cm -A"
	//actions = append(actions, events.Action{
	//	DisplayName: "Sample",
	//	Command:          fmt.Sprintf("%s %s", universalBotNamePlaceholder, cmd),
	//	ExecutorBindings: []string{"kubectl-read-only"},
	//})

	return actions, nil
}

// ExecuteEventAction executes action for given event.
// WARNING: The result interactive.Message contains BotNamePlaceholder, which should be replaced before sending the message.
func (p *Provider) ExecuteEventAction(ctx context.Context, action events.Action) interactive.GenericMessage {
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
		Message:         strings.TrimSpace(strings.TrimPrefix(action.Command, universalBotNamePlaceholder)),
		User:            fmt.Sprintf("Automation %q", action.DisplayName),
	})
	response := e.Execute(ctx)

	return &genericMessage{response: response}
}

type genericMessage struct {
	response interactive.Message
}

// ForBot returns message prepared for a bot with a given name.
func (g *genericMessage) ForBot(botName string) interactive.Message {
	g.response.ReplaceBotNameInCommands(universalBotNamePlaceholder, botName)
	return g.response
}

type universalNotifierHandler struct{}

func (n *universalNotifierHandler) NotificationsEnabled(_ string) bool {
	return false
}

func (n *universalNotifierHandler) SetNotificationsEnabled(_ string, _ bool) error {
	return errors.New("setting notification from automated action is not supported. Use Botkube commands on a specific channel to set notifications")
}

func (n *universalNotifierHandler) BotName() string {
	return universalBotNamePlaceholder
}
