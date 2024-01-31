package builder_test

import (
	"context"
	"errors"
	"testing"

	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeshop/botkube/internal/command"
	"github.com/kubeshop/botkube/internal/executor/kubectl/builder"
	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/execute/kubectl"
	"github.com/kubeshop/botkube/pkg/loggerx"
)

const (
	testingBotName = "@BKTesting"
	blockID        = "dropdown-block-id-403aca17d958"
)

func TestCommandPreview(t *testing.T) {
	tests := []struct {
		name string
		cmd  string

		expMsg api.Message
	}{
		{
			name: "Print all dropdowns and full command on verb change",
			cmd:  "@builder --verbs",

			expMsg: fixStateBuilderMessage("kubectl get pods nginx2 -n default", "@BKTesting kubectl get pods nginx2 -n default", fixAllDropdown(true)...),
		},
		{
			name: "Print all dropdowns and command without the resource name on resource type change",
			cmd:  "@builder --resource-type",

			expMsg: fixStateBuilderMessage("kubectl get pods -n default", "@BKTesting kubectl get pods -n default", fixAllDropdown(false)...),
		},
		{
			name: "Print all dropdowns and full command on resource name change",
			cmd:  "@builder --resource-name",

			expMsg: fixStateBuilderMessage("kubectl get pods nginx2 -n default", "@BKTesting kubectl get pods nginx2 -n default", fixAllDropdown(true)...),
		},
		{
			name: "Print all dropdowns and command without the resource name on namespace change",
			cmd:  "@builder --namespace",

			expMsg: fixStateBuilderMessage("kubectl get pods -n default", "@BKTesting kubectl get pods -n default", fixAllDropdown(false)...),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			var (
				expKubectlCmd = `get pods --ignore-not-found=true -o go-template='{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}' -n default`
				defaultNS     = "default"
				state         = fixStateForAllDropdowns()
				kcExecutor    = &fakeKcExecutor{}
				nsLister      = &fakeNamespaceLister{}
				authCheck     = &fakeAuthChecker{}
			)

			kcCmdBuilder := builder.NewKubectl(kcExecutor, builder.Config{
				Allowed: builder.AllowedResources{
					Verbs:     []string{"describe", "get"},
					Resources: []string{"deployments", "pods"},
				},
			}, loggerx.NewNoop(), kubectl.NewFakeCommandGuard(), defaultNS, nsLister, authCheck)
			// when
			gotMsg, err := kcCmdBuilder.Handle(context.Background(), tc.cmd, true, state)
			gotMsg.ReplaceBotNamePlaceholder(testingBotName)

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.expMsg, gotMsg)
			assert.Equal(t, expKubectlCmd, kcExecutor.command)
			assert.Equal(t, defaultNS, kcExecutor.defaultNamespace)
		})
	}
}

func TestNonInteractivePlatform(t *testing.T) {
	// given
	kcCmdBuilder := builder.NewKubectl(nil, builder.Config{}, loggerx.NewNoop(), nil, "defaultNS", nil, nil)

	// when
	gotMsg, err := kcCmdBuilder.Handle(context.Background(), "@builder", false, nil)

	// then
	require.NoError(t, err)
	assert.Equal(t, api.Message{
		Type: api.BaseBodyWithFilterMessage,
		BaseBody: api.Body{
			Plaintext: "Please specify the kubectl command",
		},
	}, gotMsg)
}

func TestShouldReturnInitialMessage(t *testing.T) {
	// given
	kcCmdBuilder := builder.NewKubectl(nil, builder.Config{
		Allowed: builder.AllowedResources{
			Verbs:     []string{"describe", "get"},
			Resources: []string{"deployments", "pods"},
		},
	}, loggerx.NewNoop(), kubectl.NewFakeCommandGuard(), "defaultNS", nil, nil)
	expMsg := fixInitialBuilderMessage()

	// when
	cmd := "@builder"
	gotMsg, err := kcCmdBuilder.Handle(context.Background(), cmd, true, nil)
	gotMsg.ReplaceBotNamePlaceholder(testingBotName)

	// then
	require.NoError(t, err)

	require.Len(t, gotMsg.Sections, 1)
	assert.NotEmpty(t, gotMsg.Sections[0].Selects.ID) // assert that we fill that property
	gotMsg.Sections[0].Selects.ID = ""                // zero that before comparison, as this is UUID that it's different in each test execution.

	assert.Equal(t, expMsg, gotMsg)
}

func TestShouldNotPrintTheResourceNameIfKubectlExecutorFails(t *testing.T) {
	// given
	var (
		state      = fixStateForAllDropdowns()
		kcExecutor = &fakeErrorKcExecutor{}
		nsLister   = &fakeNamespaceLister{}
		authCheck  = &fakeAuthChecker{}
		cmd        = "@builder --verbs"
		expMsg     = fixStateBuilderMessage("kubectl get pods -n default", "@BKTesting kubectl get pods -n default", fixVerbsDropdown(), fixResourceTypeDropdown(), fixEmptyResourceNamesDropdown(), fixNamespaceDropdown())
	)

	kcCmdBuilder := builder.NewKubectl(kcExecutor, builder.Config{
		Allowed: builder.AllowedResources{
			Verbs:     []string{"describe", "get"},
			Resources: []string{"deployments", "pods"},
		},
	}, loggerx.NewNoop(), kubectl.NewFakeCommandGuard(), "default", nsLister, authCheck)

	// when
	gotMsg, err := kcCmdBuilder.Handle(context.Background(), cmd, true, state)
	gotMsg.ReplaceBotNamePlaceholder(testingBotName)

	// then
	require.NoError(t, err)
	assert.Equal(t, expMsg, gotMsg)
}

func TestShouldPrintErrMessageIfUserHasInsufficientPerms(t *testing.T) {
	// given
	var (
		state      = fixStateForAllDropdowns()
		kcExecutor = &fakeKcExecutor{}
		nsLister   = &fakeNamespaceLister{}
		authCheck  = &fakeAuthChecker{fixErr: errors.New("not enough permissions")}
		guard      = kubectl.NewFakeCommandGuard()
		cmd        = "@builder --verbs"
		expMsg     = fixInsufficientPermsMessage(fixAllDropdown(true)...)
	)

	kcCmdBuilder := builder.NewKubectl(kcExecutor, builder.Config{
		Allowed: builder.AllowedResources{
			Verbs:     []string{"describe", "get"},
			Resources: []string{"deployments", "pods"},
		},
	}, loggerx.NewNoop(), guard, "default", nsLister, authCheck)

	// when
	gotMsg, err := kcCmdBuilder.Handle(context.Background(), cmd, true, state)
	gotMsg.ReplaceBotNamePlaceholder(testingBotName)

	// then
	require.NoError(t, err)

	assert.Equal(t, expMsg, gotMsg)
}

func fixInsufficientPermsMessage(dropdowns ...api.Select) api.Message {
	return api.Message{
		Sections: []api.Section{
			{
				Selects: api.Selects{
					ID:    blockID, // It's important to have the same ID as we have in fixture state object.
					Items: dropdowns,
				},
			},
			{
				Base: api.Base{
					Header: "Missing permissions",
					Body: api.Body{
						Plaintext: "not enough permissions",
					},
				},
				Context: api.ContextItems{
					api.ContextItem{
						Text: "To learn more about `kubectl` RBAC visit https://docs.botkube.io/configuration/executor/kubectl.",
					},
				},
			},
		},
		OnlyVisibleForYou: true,
		ReplaceOriginal:   true,
	}
}

func TestShouldPrintErrMessageIfGuardFails(t *testing.T) {
	// given
	var (
		guardErr   = errors.New("internal guard err")
		state      = fixStateNotAllowedVerbDropdown()
		kcExecutor = &fakeKcExecutor{}
		nsLister   = &fakeNamespaceLister{}
		guard      = &fakeErrCommandGuard{fixErr: guardErr}
		cmd        = "@builder --verbs"
	)

	kcCmdBuilder := builder.NewKubectl(kcExecutor, builder.Config{
		Allowed: builder.AllowedResources{
			Verbs:     []string{"describe", "get", "exec"},
			Resources: []string{"deployments", "pods"},
		},
	}, loggerx.NewNoop(), guard, "default", nsLister, nil)

	// when
	_, err := kcCmdBuilder.Handle(context.Background(), cmd, true, state)

	// then
	require.EqualError(t, err, guardErr.Error())
}

func TestShouldPrintErrMessageIfVerbNotAllowed(t *testing.T) {
	// given
	var (
		state      = fixStateNotAllowedVerbDropdown()
		kcExecutor = &fakeKcExecutor{}
		nsLister   = &fakeNamespaceLister{}
		guard      = &fakeErrCommandGuard{fixErr: command.ErrVerbNotSupported}
		cmd        = "@builder --verbs"
		expMsg     = fixNotSupportedVerbMessage()
	)

	kcCmdBuilder := builder.NewKubectl(kcExecutor, builder.Config{
		Allowed: builder.AllowedResources{
			Verbs:     []string{"describe", "get", "exec"},
			Resources: []string{"deployments", "pods"},
		},
	}, loggerx.NewNoop(), guard, "default", nsLister, nil)

	// when
	gotMsg, err := kcCmdBuilder.Handle(context.Background(), cmd, true, state)
	gotMsg.ReplaceBotNamePlaceholder(testingBotName)

	// then
	require.NoError(t, err)

	require.Len(t, gotMsg.Sections, 2)
	assert.NotEmpty(t, gotMsg.Sections[0].Selects.ID) // assert that we fill that property
	gotMsg.Sections[0].Selects.ID = ""                // zero that before comparison, as this is UUID that it's different in each test execution.

	assert.Equal(t, expMsg, gotMsg)
}

func fixNotSupportedVerbMessage() api.Message {
	return api.Message{
		Sections: []api.Section{
			{
				Selects: api.Selects{
					Items: []api.Select{
						{
							Name:    "Select command",
							Command: "@BKTesting kubectl @builder --verbs",
							OptionGroups: []api.OptionGroup{
								{
									Name: "Select command",
									Options: []api.OptionItem{
										{
											Name:  "describe",
											Value: "describe",
										},
										{
											Name:  "get",
											Value: "get",
										},
										{
											Name:  "exec",
											Value: "exec",
										},
									},
								},
							},
						},
					},
				},
			},
			{
				Base: api.Base{
					Body: api.Body{
						Plaintext: `:exclamation: Unfortunately, interactive command builder doesn't support "exec" verb yet.`,
					},
				},
			},
		},
		OnlyVisibleForYou: true,
		ReplaceOriginal:   true,
	}
}

func fixStateForAllDropdowns() *slack.BlockActionStates {
	return &slack.BlockActionStates{
		Values: map[string]map[string]slack.BlockAction{
			blockID: {
				"kubectl @builder --resource-name": {
					SelectedOption: slack.OptionBlockObject{
						Value: "nginx2",
					},
				},
				"kubectl @builder --resource-type": slack.BlockAction{
					SelectedOption: slack.OptionBlockObject{
						Value: "pods",
					},
				},
				"kubectl @builder --verbs": slack.BlockAction{
					SelectedOption: slack.OptionBlockObject{
						Value: "get",
					},
				},
			},
		},
	}
}

func fixStateNotAllowedVerbDropdown() *slack.BlockActionStates {
	return &slack.BlockActionStates{
		Values: map[string]map[string]slack.BlockAction{
			blockID: {
				"kubectl @builder --verbs": slack.BlockAction{
					SelectedOption: slack.OptionBlockObject{
						Value: "exec",
					},
				},
			},
		},
	}
}

func fixInitialBuilderMessage() api.Message {
	verbsDropdown := fixVerbsDropdown()
	verbsDropdown.InitialOption = nil // initial message shouldn't have anything selected.
	return api.Message{
		Sections: []api.Section{
			{
				Selects: api.Selects{
					Items: []api.Select{
						verbsDropdown,
					},
				},
			},
		},
		OnlyVisibleForYou: true,
		ReplaceOriginal:   false,
	}
}

func fixVerbsDropdown() api.Select {
	return api.Select{
		Name:    "Select command",
		Command: "@BKTesting kubectl @builder --verbs",
		InitialOption: &api.OptionItem{
			Name:  "get",
			Value: "get",
		},
		OptionGroups: []api.OptionGroup{
			{
				Name: "Select command",
				Options: []api.OptionItem{
					{
						Name:  "describe",
						Value: "describe",
					},
					{
						Name:  "get",
						Value: "get",
					},
				},
			},
		},
	}
}

func fixResourceTypeDropdown() api.Select {
	return api.Select{
		Name:    "Select resource",
		Command: "@BKTesting kubectl @builder --resource-type",
		InitialOption: &api.OptionItem{
			Name:  "pods",
			Value: "pods",
		},
		OptionGroups: []api.OptionGroup{
			{
				Name: "Select resource",
				Options: []api.OptionItem{
					{
						Name:  "deployments",
						Value: "deployments",
					},
					{
						Name:  "pods",
						Value: "pods",
					},
				},
			},
		},
	}
}

func fixNamespaceDropdown() api.Select {
	return api.Select{
		Name:    "Select namespace",
		Command: "@BKTesting kubectl @builder --namespace",
		OptionGroups: []api.OptionGroup{
			{
				Name: "Select namespace",
				Options: []api.OptionItem{
					{
						Name:  "default (namespace)",
						Value: "default",
					},
				},
			},
		},
		InitialOption: &api.OptionItem{
			Name:  "default (namespace)",
			Value: "default",
		},
	}
}

func fixEmptyResourceNamesDropdown() api.Select {
	return api.Select{
		Name: "No resources found",
		Type: api.ExternalSelect,
		InitialOption: &api.OptionItem{
			Name:  "No resources found",
			Value: "no-resources",
		},
	}
}

func fixResourceNamesDropdown(includeInitialOpt bool) api.Select {
	var opt *api.OptionItem
	if includeInitialOpt {
		opt = &api.OptionItem{
			Name:  "nginx2",
			Value: "nginx2",
		}
	}

	return api.Select{
		Name:          "Select resource name",
		Command:       "@BKTesting kubectl @builder --resource-name",
		InitialOption: opt,
		OptionGroups: []api.OptionGroup{
			{
				Name: "Select resource name",
				Options: []api.OptionItem{
					{
						Name:  "nginx2",
						Value: "nginx2",
					},
					{
						Name:  "grafana",
						Value: "grafana",
					},
					{
						Name:  "argo",
						Value: "argo",
					},
				},
			},
		},
	}
}

func fixAllDropdown(includeResourceName bool) []api.Select {
	return []api.Select{
		fixVerbsDropdown(),
		fixResourceTypeDropdown(),
		fixResourceNamesDropdown(includeResourceName),
		fixNamespaceDropdown(),
	}
}

func fixStateBuilderMessage(kcCommandPreview, kcCommand string, dropdowns ...api.Select) api.Message {
	return api.Message{
		Sections: []api.Section{
			{
				Selects: api.Selects{
					ID:    blockID, // It's important to have the same ID as we have in fixture state object.
					Items: dropdowns,
				},
			},
			{
				Base: api.Base{
					Body: api.Body{
						CodeBlock: kcCommandPreview,
					},
				},
				PlaintextInputs: api.LabelInputs{
					api.LabelInput{
						Command:          "@BKTesting kubectl @builder --filter-query ",
						DispatchedAction: api.DispatchInputActionOnCharacter,
						Text:             "Filter output",
						Placeholder:      "Filter output by string (optional)",
					},
				},
			},
			{
				Buttons: api.Buttons{
					api.Button{
						Name:    "Run command",
						Command: kcCommand,
						Style:   "primary",
					},
				},
			},
		},
		OnlyVisibleForYou: true,
		ReplaceOriginal:   true,
	}
}

type fakeKcExecutor struct {
	command          string
	defaultNamespace string
}

func (r *fakeKcExecutor) RunKubectlCommand(_ context.Context, defaultNamespace, cmd string) (string, error) {
	r.defaultNamespace = defaultNamespace
	r.command = cmd

	return "nginx2\ngrafana\nargo", nil
}

type fakeErrorKcExecutor struct{}

func (r *fakeErrorKcExecutor) RunKubectlCommand(context.Context, string, string) (string, error) {
	return "", errors.New("fake error")
}

type fakeNamespaceLister struct{}

func (f *fakeNamespaceLister) List(_ context.Context, _ metav1.ListOptions) (*corev1.NamespaceList, error) {
	return &corev1.NamespaceList{
		Items: []corev1.Namespace{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name: "default",
				},
			},
		},
	}, nil
}

type fakeAuthChecker struct {
	fixErr error
}

func (r *fakeAuthChecker) CheckUserAccess(ns, verb, resource, name string) error {
	return r.fixErr
}

type fakeErrCommandGuard struct {
	fixErr error
}

// FilterSupportedVerbs filters out unsupported verbs by the interactive commands.
func (f *fakeErrCommandGuard) FilterSupportedVerbs(allVerbs []string) []string {
	return allVerbs
}

// GetAllowedResourcesForVerb returns allowed resources types for a given verb.
func (f *fakeErrCommandGuard) GetAllowedResourcesForVerb(string, []string) ([]command.Resource, error) {
	return nil, f.fixErr
}

// GetResourceDetails returns resource details.
func (f *fakeErrCommandGuard) GetResourceDetails(string, string) (command.Resource, error) {
	return command.Resource{}, f.fixErr
}
