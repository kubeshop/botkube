package execute_test

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kubeshop/botkube/internal/loggerx"
	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/bot/interactive"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/execute"
	"github.com/kubeshop/botkube/pkg/execute/kubectl"
)

const testingBotName = "@BKTesting"

func TestCommandPreview(t *testing.T) {
	fixBindings := []string{"kc-read-only", "kc-delete-pod"}

	tests := []struct {
		name string
		args []string

		expMsg interactive.Message
	}{
		{
			name: "Print all dropdowns and full command on verb change",
			args: strings.Fields("kc-cmd-builder --verbs"),

			expMsg: fixStateBuilderMessage("kubectl get pods nginx2 -n default", "@BKTesting kubectl get pods nginx2 -n default", fixAllDropdown(true)...),
		},
		{
			name: "Print all dropdowns and command without the resource name on resource type change",
			args: strings.Fields("kc-cmd-builder --resource-type"),

			expMsg: fixStateBuilderMessage("kubectl get pods -n default", "@BKTesting kubectl get pods -n default", fixAllDropdown(false)...),
		},
		{
			name: "Print all dropdowns and full command on resource name change",
			args: strings.Fields("kc-cmd-builder --resource-name"),

			expMsg: fixStateBuilderMessage("kubectl get pods nginx2 -n default", "@BKTesting kubectl get pods nginx2 -n default", fixAllDropdown(true)...),
		},
		{
			name: "Print all dropdowns and command without the resource name on namespace change",
			args: strings.Fields("kc-cmd-builder --namespace"),

			expMsg: fixStateBuilderMessage("kubectl get pods -n default", "@BKTesting kubectl get pods -n default", fixAllDropdown(false)...),
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			var (
				expKubectlCmd = `kubectl get pods --ignore-not-found=true -o go-template='{{range .items}}{{.metadata.name}}{{"\n"}}{{end}}' -n default`
				state         = fixStateForAllDropdowns()
				kcExecutor    = &fakeKcExecutor{}
				nsLister      = &fakeNamespaceLister{}
				kcMerger      = newFakeKcMerger([]string{"get", "describe"}, []string{"deployments", "pods"})
			)

			kcCmdBuilderExecutor := execute.NewKubectlCmdBuilder(loggerx.NewNoop(), kcMerger, kcExecutor, nsLister, &FakeCommandGuard{})

			// when
			gotMsg, err := kcCmdBuilderExecutor.Do(context.Background(), tc.args, config.SocketSlackCommPlatformIntegration, fixBindings, state, testingBotName, "header", execute.CommandContext{})

			// then
			require.NoError(t, err)
			assert.Equal(t, tc.expMsg, gotMsg)
			assert.Equal(t, expKubectlCmd, kcExecutor.command)
			assert.True(t, kcExecutor.isAuthed)
			assert.Equal(t, fixBindings, kcExecutor.bindings)
		})
	}
}

func TestCommandBuilderCanHandleAndGetPrefix(t *testing.T) {
	tests := []struct {
		name string
		args []string

		expPrefix    string
		expCanHandle bool
	}{
		{
			name: "Dropdown verbs",
			args: strings.Fields("kc-cmd-builder --verbs my-verb"),

			expCanHandle: true,
			expPrefix:    "kc-cmd-builder --verbs",
		},
		{
			name: "Dropdown resource type",
			args: strings.Fields("kc-cmd-builder --resource-type my-resource-type"),

			expCanHandle: true,
			expPrefix:    "kc-cmd-builder --resource-type",
		},
		{
			name: "Dropdown resource name",
			args: strings.Fields("kc-cmd-builder --resource-name my-resource-name"),

			expCanHandle: true,
			expPrefix:    "kc-cmd-builder --resource-name",
		},
		{
			name: "Dropdown namespace",
			args: strings.Fields("kc-cmd-builder --namespace my-namespace"),

			expCanHandle: true,
			expPrefix:    "kc-cmd-builder --namespace",
		},
		{
			name: "Dropdown namespace",
			args: strings.Fields("kc-cmd-builder --namespace my-namespace other-arg-but-we-dont-care"),

			expCanHandle: true,
			expPrefix:    "kc-cmd-builder --namespace",
		},
		{
			name: "Kubectl full command",
			args: strings.Fields("kubectl"),

			expCanHandle: true,
			expPrefix:    "kubectl",
		},
		{
			name: "Kubectl full command",
			args: strings.Fields("kubectl get pod"),

			expCanHandle: false,
			expPrefix:    "",
		},
		{
			name: "Unknown command",
			args: strings.Fields("helm"),

			expCanHandle: false,
			expPrefix:    "",
		},
		{
			name: "Wrong command",
			args: strings.Fields("kc-cmd-builder"),

			expCanHandle: false,
			expPrefix:    "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// given
			kcCmdBuilderExecutor := execute.NewKubectlCmdBuilder(nil, nil, nil, nil, &FakeCommandGuard{})

			// when
			gotCanHandle := kcCmdBuilderExecutor.CanHandle(tc.args)
			gotPrefix := kcCmdBuilderExecutor.GetCommandPrefix(tc.args)

			// then
			assert.Equal(t, tc.expCanHandle, gotCanHandle)
			assert.Equal(t, tc.expPrefix, gotPrefix)
		})
	}
}

func TestErrorUserMessageOnPlatformsOtherThanSocketSlack(t *testing.T) {
	platforms := []config.CommPlatformIntegration{
		config.SlackCommPlatformIntegration,
		config.MattermostCommPlatformIntegration,
		config.TeamsCommPlatformIntegration,
		config.DiscordCommPlatformIntegration,
		config.ElasticsearchCommPlatformIntegration,
		config.WebhookCommPlatformIntegration,
	}
	for _, platform := range platforms {
		t.Run(fmt.Sprintf("Should ignore %s", platform), func(t *testing.T) {
			// given
			const cmdHeader = "header"
			kcCmdBuilderExecutor := execute.NewKubectlCmdBuilder(loggerx.NewNoop(), nil, nil, nil, nil)

			// when
			gotMsg, err := kcCmdBuilderExecutor.Do(context.Background(), []string{"kc"}, platform, nil, nil, "", cmdHeader, execute.CommandContext{})

			// then
			require.NoError(t, err)
			assert.Equal(t, interactive.Message{
				Description: cmdHeader,
				Message: api.Message{
					BaseBody: api.Body{
						Plaintext: "Please specify the kubectl command",
					},
				},
			}, gotMsg)
		})
	}
}

func TestShouldReturnInitialMessage(t *testing.T) {
	// given
	var (
		kcMerger             = newFakeKcMerger([]string{"get", "describe"}, []string{"deployments", "pods"})
		kcCmdBuilderExecutor = execute.NewKubectlCmdBuilder(loggerx.NewNoop(), kcMerger, nil, nil, &FakeCommandGuard{})
		expMsg               = fixInitialBuilderMessage()
	)

	// when command args are not specified
	cmd := []string{"kc-cmd-builder"}
	gotMsg, err := kcCmdBuilderExecutor.Do(context.Background(), cmd, config.SocketSlackCommPlatformIntegration, nil, nil, testingBotName, "cmdHeader", execute.CommandContext{})

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
		kcMerger   = newFakeKcMerger([]string{"get", "describe"}, []string{"deployments", "pods"})
		args       = []string{"kc-cmd-builder", "--verbs"}
		expMsg     = fixStateBuilderMessage("kubectl get pods -n default", "@BKTesting kubectl get pods -n default", fixVerbsDropdown(), fixResourceTypeDropdown(), fixEmptyResourceNamesDropdown(), fixNamespaceDropdown())
	)

	kcCmdBuilderExecutor := execute.NewKubectlCmdBuilder(loggerx.NewNoop(), kcMerger, kcExecutor, nsLister, &FakeCommandGuard{})

	// when
	gotMsg, err := kcCmdBuilderExecutor.Do(context.Background(), args, config.SocketSlackCommPlatformIntegration, []string{"kc-read-only"}, state, testingBotName, "header", execute.CommandContext{})

	// then
	require.NoError(t, err)
	assert.Equal(t, expMsg, gotMsg)
}

func fixStateForAllDropdowns() *slack.BlockActionStates {
	return &slack.BlockActionStates{
		Values: map[string]map[string]slack.BlockAction{
			"dropdown-block-id-403aca17d958": {
				"@BKTesting kc-cmd-builder --resource-name": {
					SelectedOption: slack.OptionBlockObject{
						Value: "nginx2",
					},
				},
				"@BKTesting kc-cmd-builder --resource-type": slack.BlockAction{
					SelectedOption: slack.OptionBlockObject{
						Value: "pods",
					},
				},
				"@BKTesting kc-cmd-builder --verbs": slack.BlockAction{
					SelectedOption: slack.OptionBlockObject{
						Value: "get",
					},
				},
			},
		},
	}
}

func fixInitialBuilderMessage() interactive.Message {
	verbsDropdown := fixVerbsDropdown()
	verbsDropdown.InitialOption = nil // initial message shouldn't have anything selected.
	return interactive.Message{
		Message: api.Message{
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
		},
	}
}

func fixVerbsDropdown() api.Select {
	return api.Select{
		Name:    "Select command",
		Command: "@BKTesting kc-cmd-builder --verbs",
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
		Command: "@BKTesting kc-cmd-builder --resource-type",
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
		Command: "@BKTesting kc-cmd-builder --namespace",
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
		Command:       "@BKTesting kc-cmd-builder --resource-name",
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

func fixStateBuilderMessage(kcCommandPreview, kcCommand string, dropdowns ...api.Select) interactive.Message {
	return interactive.Message{
		Message: api.Message{
			Sections: []api.Section{
				{
					Selects: api.Selects{
						ID:    "dropdown-block-id-403aca17d958", // It's important to have the same ID as we have in fixture state object.
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
							Command:          "@BKTesting kc-cmd-builder --filter-query ",
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
		},
	}
}

type fakeKcExecutor struct {
	isAuthed bool
	command  string
	bindings []string
}

func (r *fakeKcExecutor) Execute(bindings []string, command string, isAuthChannel bool, _ execute.CommandContext) (string, error) {
	r.bindings = bindings
	r.command = command
	r.isAuthed = isAuthChannel

	return "nginx2\ngrafana\nargo", nil
}

type fakeErrorKcExecutor struct{}

func (r *fakeErrorKcExecutor) Execute(_ []string, _ string, _ bool, _ execute.CommandContext) (string, error) {
	return "", errors.New("fake error")
}

type fakeKcMerger struct {
	allowedVerbs     []string
	allowedResources []string
}

func newFakeKcMerger(allowedVerbs []string, allowedResources []string) *fakeKcMerger {
	return &fakeKcMerger{allowedVerbs: allowedVerbs, allowedResources: allowedResources}
}

func (r *fakeKcMerger) MergeAllEnabled(_ []string) kubectl.EnabledKubectl {
	verbs := map[string]struct{}{}
	for _, name := range r.allowedVerbs {
		verbs[name] = struct{}{}
	}
	resources := map[string]struct{}{}
	for _, name := range r.allowedResources {
		resources[name] = struct{}{}
	}
	resourceNamespaces := map[string]config.RegexConstraints{}
	for _, name := range r.allowedResources {
		resourceNamespaces[name] = config.RegexConstraints{
			Include: []string{"default"},
		}
	}
	return kubectl.EnabledKubectl{
		AllowedKubectlVerb:           verbs,
		AllowedKubectlResource:       resources,
		AllowedNamespacesPerResource: resourceNamespaces,
	}
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
