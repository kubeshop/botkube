package config_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/MakeNowJust/heredoc"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"

	"github.com/kubeshop/botkube/pkg/config"
)

func TestPersistenceManager_PersistSourceBindings(t *testing.T) {
	// given
	commGroupName := "default-group"
	cfg := config.PartialPersistentConfig{
		ConfigMap: config.K8sResourceRef{
			Name:      "foo",
			Namespace: "ns",
		},
		FileName: "_runtime_state.yaml",
	}

	testCases := []struct {
		Name                string
		InputCfgMap         *v1.ConfigMap
		InputPlatform       config.CommPlatformIntegration
		InputChannel        string
		InputSourceBindings []string
		ExpectedErrMessage  string
		Expected            *v1.ConfigMap
	}{
		{
			Name:                "Empty state files",
			InputPlatform:       config.DiscordCommPlatformIntegration,
			InputChannel:        "foo",
			InputSourceBindings: []string{"first", "second"},
			InputCfgMap: &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cfg.ConfigMap.Name,
					Namespace: cfg.ConfigMap.Namespace,
				},
				Data: map[string]string{
					cfg.FileName: "",
				},
			},
			Expected: &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cfg.ConfigMap.Name,
					Namespace: cfg.ConfigMap.Namespace,
				},
				Data: map[string]string{
					cfg.FileName: heredoc.Doc(`
                      communications:
                        default-group:
                          discord:
                            channels:
                              foo:
                                bindings:
                                  sources:
                                    - first
                                    - second
					`),
				},
			},
		},
		{
			Name:                "Empty state files - MS Teams",
			InputPlatform:       config.TeamsCommPlatformIntegration,
			InputChannel:        "foo",
			InputSourceBindings: []string{"first", "second"},
			InputCfgMap: &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cfg.ConfigMap.Name,
					Namespace: cfg.ConfigMap.Namespace,
				},
				Data: map[string]string{
					cfg.FileName: "",
				},
			},
			Expected: &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cfg.ConfigMap.Name,
					Namespace: cfg.ConfigMap.Namespace,
				},
				Data: map[string]string{
					cfg.FileName: heredoc.Doc(`
                      communications:
                        default-group:
                          teams:
                            bindings:
                              sources:
                                - first
                                - second
					`),
				},
			},
		},
		{
			Name:                "Existing state files",
			InputChannel:        "general",
			InputPlatform:       config.SlackCommPlatformIntegration,
			InputSourceBindings: []string{"new", "newer"},
			InputCfgMap: &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cfg.ConfigMap.Name,
					Namespace: cfg.ConfigMap.Namespace,
				},
				Data: map[string]string{
					cfg.FileName: heredoc.Doc(`
                      communications:
                        default-group:
                          slack:
                            channels:
                              foo:
                                bindings:
                                  sources:
                                    - foo
                                    - bar
                              general:
                                bindings:
                                  sources:
                                    - old
                                    - older
                                    - oldest
					`),
				},
			},
			Expected: &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cfg.ConfigMap.Name,
					Namespace: cfg.ConfigMap.Namespace,
				},
				Data: map[string]string{
					cfg.FileName: heredoc.Doc(`
                      communications:
                        default-group:
                          slack:
                            channels:
                              foo:
                                bindings:
                                  sources:
                                    - foo
                                    - bar
                              general:
                                bindings:
                                  sources:
                                    - new
                                    - newer
					`),
				},
			},
		},
		{
			Name:                "Existing state files - MS Teams",
			InputChannel:        "anything",
			InputPlatform:       config.TeamsCommPlatformIntegration,
			InputSourceBindings: []string{"new", "newer"},
			InputCfgMap: &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cfg.ConfigMap.Name,
					Namespace: cfg.ConfigMap.Namespace,
				},
				Data: map[string]string{
					cfg.FileName: heredoc.Doc(`
                      communications:
                        default-group:
                          slack:
                            channels:
                              foo:
                                bindings:
                                  sources:
                                    - foo
                                    - bar
                              general:
                                bindings:
                                  sources:
                                    - old
                                    - older
                                    - oldest
                          teams:
                            bindings:
                              sources:
                                - old
                                - older
                                - oldest               
					`),
				},
			},
			Expected: &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cfg.ConfigMap.Name,
					Namespace: cfg.ConfigMap.Namespace,
				},
				Data: map[string]string{
					cfg.FileName: heredoc.Doc(`
                      communications:
                        default-group:
                          slack:
                            channels:
                              foo:
                                bindings:
                                  sources:
                                    - foo
                                    - bar
                              general:
                                bindings:
                                  sources:
                                    - old
                                    - older
                                    - oldest
                          teams:
                            bindings:
                              sources:
                                - new
                                - newer
					`),
				},
			},
		},
		{
			Name:                "Unsupported platform",
			InputPlatform:       config.WebhookCommPlatformIntegration,
			InputChannel:        "foo",
			InputSourceBindings: []string{"source"},
			InputCfgMap: &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cfg.ConfigMap.Name,
					Namespace: cfg.ConfigMap.Namespace,
				},
			},
			ExpectedErrMessage: `unsupported platform to persist data`,
		},
		{
			Name:                "No ConfigMap",
			InputChannel:        "foo",
			InputPlatform:       config.SlackCommPlatformIntegration,
			InputSourceBindings: []string{"source"},
			InputCfgMap:         &v1.ConfigMap{},
			ExpectedErrMessage:  `while getting the ConfigMap: configmaps "foo" not found`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			logger, _ := logtest.NewNullLogger()
			k8sCli := fake.NewSimpleClientset(testCase.InputCfgMap)
			manager := config.NewManager(logger, config.PersistentConfig{Runtime: cfg}, k8sCli)

			// when
			err := manager.PersistSourceBindings(context.Background(), commGroupName, testCase.InputPlatform, testCase.InputChannel, testCase.InputSourceBindings)

			// then
			if testCase.ExpectedErrMessage != "" {
				require.Error(t, err)
				assert.EqualError(t, err, testCase.ExpectedErrMessage)
				return
			}

			require.NoError(t, err)

			cfgMap, err := k8sCli.CoreV1().ConfigMaps(cfg.ConfigMap.Namespace).Get(context.Background(), cfg.ConfigMap.Name, metav1.GetOptions{})
			require.NoError(t, err)
			assert.Equal(t, testCase.Expected, cfgMap)
		})
	}
}

func TestPersistenceManager_PersistNotificationsEnabled(t *testing.T) {
	// given
	commGroupName := "default-group"
	cfg := config.PartialPersistentConfig{
		ConfigMap: config.K8sResourceRef{
			Name:      "foo",
			Namespace: "ns",
		},
		FileName: "__startup_state.yaml",
	}

	testCases := []struct {
		Name               string
		InputCfgMap        *v1.ConfigMap
		InputPlatform      config.CommPlatformIntegration
		InputChannel       string
		InputEnabled       bool
		ExpectedErrMessage string
		Expected           *v1.ConfigMap
	}{
		{
			Name:          "Empty state files",
			InputPlatform: config.DiscordCommPlatformIntegration,
			InputChannel:  "foo",
			InputEnabled:  false,
			InputCfgMap: &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cfg.ConfigMap.Name,
					Namespace: cfg.ConfigMap.Namespace,
				},
				Data: map[string]string{
					cfg.FileName: "",
				},
			},
			Expected: &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cfg.ConfigMap.Name,
					Namespace: cfg.ConfigMap.Namespace,
				},
				Data: map[string]string{
					cfg.FileName: heredoc.Doc(`
                      communications:
                        default-group:
                          discord:
                            channels:
                              foo:
                                notification:
                                  disabled: true
					`),
				},
			},
		},
		{
			Name:          "Existing state files",
			InputChannel:  "general",
			InputPlatform: config.SlackCommPlatformIntegration,
			InputEnabled:  true,
			InputCfgMap: &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cfg.ConfigMap.Name,
					Namespace: cfg.ConfigMap.Namespace,
				},
				Data: map[string]string{
					cfg.FileName: heredoc.Doc(`
                      communications:
                        default-group:
                          slack:
                            channels:
                              foo:
                                notification:
                                  disabled: true
                              general:
                                notification:
                                  disabled: true
                      filters:
                        kubernetes:
                          objectAnnotationChecker: true
                          nodeEventsChecker: true
					`),
				},
			},
			Expected: &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cfg.ConfigMap.Name,
					Namespace: cfg.ConfigMap.Namespace,
				},
				Data: map[string]string{
					cfg.FileName: heredoc.Doc(`
                      communications:
                        default-group:
                          slack:
                            channels:
                              foo:
                                notification:
                                  disabled: true
                              general:
                                notification:
                                  disabled: false
                      filters:
                        kubernetes:
                          objectAnnotationChecker: true
                          nodeEventsChecker: true
					`),
				},
			},
		},
		{
			Name:          "Unsupported platform",
			InputPlatform: config.TeamsCommPlatformIntegration,
			InputChannel:  "foo",
			InputEnabled:  false,
			InputCfgMap: &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cfg.ConfigMap.Name,
					Namespace: cfg.ConfigMap.Namespace,
				},
			},
			ExpectedErrMessage: `unsupported platform to persist data`,
		},
		{
			Name:               "No ConfigMap",
			InputChannel:       "foo",
			InputPlatform:      config.SlackCommPlatformIntegration,
			InputEnabled:       false,
			InputCfgMap:        &v1.ConfigMap{},
			ExpectedErrMessage: `while getting the ConfigMap: configmaps "foo" not found`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			logger, _ := logtest.NewNullLogger()
			k8sCli := fake.NewSimpleClientset(testCase.InputCfgMap)
			manager := config.NewManager(logger, config.PersistentConfig{Startup: cfg}, k8sCli)

			// when
			err := manager.PersistNotificationsEnabled(context.Background(), commGroupName, testCase.InputPlatform, testCase.InputChannel, testCase.InputEnabled)

			// then
			if testCase.ExpectedErrMessage != "" {
				require.Error(t, err)
				assert.EqualError(t, err, testCase.ExpectedErrMessage)
				return
			}

			require.NoError(t, err)

			cfgMap, err := k8sCli.CoreV1().ConfigMaps(cfg.ConfigMap.Namespace).Get(context.Background(), cfg.ConfigMap.Name, metav1.GetOptions{})
			require.NoError(t, err)
			assert.Equal(t, testCase.Expected, cfgMap)
		})
	}
}

func TestPersistenceManager_PersistFilterEnabled(t *testing.T) {
	// given
	cfg := config.PartialPersistentConfig{
		ConfigMap: config.K8sResourceRef{
			Name:      "foo",
			Namespace: "ns",
		},
		FileName: "__startup_state.yaml",
	}

	testCases := []struct {
		Name               string
		InputCfgMap        *v1.ConfigMap
		InputName          string
		InputEnabled       bool
		ExpectedErrMessage string
		Expected           *v1.ConfigMap
	}{
		{
			Name:         "Empty state files",
			InputName:    "ObjectAnnotationChecker",
			InputEnabled: true,
			InputCfgMap: &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cfg.ConfigMap.Name,
					Namespace: cfg.ConfigMap.Namespace,
				},
				Data: map[string]string{
					cfg.FileName: "",
				},
			},
			Expected: &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cfg.ConfigMap.Name,
					Namespace: cfg.ConfigMap.Namespace,
				},
				Data: map[string]string{
					cfg.FileName: heredoc.Doc(`
					  filters:
					    kubernetes:
					      objectAnnotationChecker: true
					      nodeEventsChecker: false
					`),
				},
			},
		},
		{
			Name:         "Existing state files",
			InputName:    "NodeEventsChecker",
			InputEnabled: false,
			InputCfgMap: &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cfg.ConfigMap.Name,
					Namespace: cfg.ConfigMap.Namespace,
				},
				Data: map[string]string{
					cfg.FileName: heredoc.Doc(`
                      communications:
                        default-group:
                          slack:
                            channels:
                              foo:
                                notification:
                                  disabled: true
                      filters:
                        kubernetes:
                          objectAnnotationChecker: true
                          nodeEventsChecker: true
					`),
				},
			},
			Expected: &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cfg.ConfigMap.Name,
					Namespace: cfg.ConfigMap.Namespace,
				},
				Data: map[string]string{
					cfg.FileName: heredoc.Doc(`
                      communications:
                        default-group:
                          slack:
                            channels:
                              foo:
                                notification:
                                  disabled: true
                      filters:
                        kubernetes:
                          objectAnnotationChecker: true
                          nodeEventsChecker: false
					`),
				},
			},
		},
		{
			Name:         "Filter not found",
			InputName:    "foo",
			InputEnabled: false,
			InputCfgMap: &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cfg.ConfigMap.Name,
					Namespace: cfg.ConfigMap.Namespace,
				},
			},
			ExpectedErrMessage: `Filter with name "foo" not found`,
		},
		{
			Name:               "No ConfigMap",
			InputName:          "foo",
			InputEnabled:       false,
			InputCfgMap:        &v1.ConfigMap{},
			ExpectedErrMessage: `while getting the ConfigMap: configmaps "foo" not found`,
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			logger, _ := logtest.NewNullLogger()
			k8sCli := fake.NewSimpleClientset(testCase.InputCfgMap)
			manager := config.NewManager(logger, config.PersistentConfig{Startup: cfg}, k8sCli)

			// when
			err := manager.PersistFilterEnabled(context.Background(), testCase.InputName, testCase.InputEnabled)

			// then
			if testCase.ExpectedErrMessage != "" {
				require.Error(t, err)
				assert.EqualError(t, err, testCase.ExpectedErrMessage)
				return
			}

			require.NoError(t, err)

			cfgMap, err := k8sCli.CoreV1().ConfigMaps(cfg.ConfigMap.Namespace).Get(context.Background(), cfg.ConfigMap.Name, metav1.GetOptions{})
			require.NoError(t, err)
			assert.Equal(t, testCase.Expected, cfgMap)
		})
	}
}

func TestPersistenceManager_ListActions(t *testing.T) {
	// given
	cfg := config.PartialPersistentConfig{
		ConfigMap: config.K8sResourceRef{
			Name:      "foo",
			Namespace: "ns",
		},
		FileName: "_runtime_state.yaml",
	}

	testCases := []struct {
		Name        string
		InputCfgMap *v1.ConfigMap
		Expected    map[string]config.Action
	}{
		{
			Name: "Empty runtime config",
			InputCfgMap: &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cfg.ConfigMap.Name,
					Namespace: cfg.ConfigMap.Namespace,
				},
			},
			Expected: map[string]config.Action{},
		},
		{
			Name: "Two actions expected",
			InputCfgMap: &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cfg.ConfigMap.Name,
					Namespace: cfg.ConfigMap.Namespace,
				},
				Data: map[string]string{
					cfg.FileName: heredoc.Doc(`
                      actions:
                        get-created-resource:
                          enabled: true
                          displayName: "get created resource"
                        get-deleted-resource:
                          enabled: false
                          displayName: "get deleted resource"
					`),
				},
			},
			Expected: map[string]config.Action{
				"get-created-resource": {Enabled: true, DisplayName: "get created resource"},
				"get-deleted-resource": {Enabled: false, DisplayName: "get deleted resource"},
			},
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			logger, _ := logtest.NewNullLogger()
			k8sCli := fake.NewSimpleClientset(testCase.InputCfgMap)
			manager := config.NewManager(logger, config.PersistentConfig{Runtime: cfg}, k8sCli)

			// when
			actions, err := manager.ListActions(context.Background())
			require.NoError(t, err)
			assert.Equal(t, testCase.Expected, actions)
		})
	}
}

func TestPersistenceManager_PersistActionEnabled(t *testing.T) {
	// given
	cfg := config.PartialPersistentConfig{
		ConfigMap: config.K8sResourceRef{
			Name:      "foo",
			Namespace: "ns",
		},
		FileName: "_runtime_state.yaml",
	}

	testCases := []struct {
		Name        string
		ActionName  string
		Enabled     bool
		Expected    map[string]config.Action
		InputCfgMap *v1.ConfigMap
		Err         error
	}{
		{
			Name:       "Action not defined",
			ActionName: "bogus",
			Enabled:    true,
			InputCfgMap: &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cfg.ConfigMap.Name,
					Namespace: cfg.ConfigMap.Namespace,
				},
			},
			Expected: map[string]config.Action{},
			Err:      fmt.Errorf("action with name \"bogus\" not found"),
		},
		{
			Name:       "Enabled switch from true to false",
			ActionName: "get-created-resource",
			Enabled:    false,
			InputCfgMap: &v1.ConfigMap{
				ObjectMeta: metav1.ObjectMeta{
					Name:      cfg.ConfigMap.Name,
					Namespace: cfg.ConfigMap.Namespace,
				},
				Data: map[string]string{
					cfg.FileName: heredoc.Doc(`
                      actions:
                        get-created-resource:
                          enabled: true
                          displayName: "get created resource"
                        get-deleted-resource:
                          enabled: false
                          displayName: "get deleted resource"
					`),
				},
			},
			Expected: map[string]config.Action{
				"get-created-resource": {Enabled: false, DisplayName: "get created resource", Bindings: config.ActionBindings{Sources: []string{}, Executors: []string{}}},
				"get-deleted-resource": {Enabled: false, DisplayName: "get deleted resource", Bindings: config.ActionBindings{Sources: []string{}, Executors: []string{}}},
			},
			Err: nil,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			logger, _ := logtest.NewNullLogger()
			k8sCli := fake.NewSimpleClientset(testCase.InputCfgMap)
			manager := config.NewManager(logger, config.PersistentConfig{Runtime: cfg}, k8sCli)

			err := manager.PersistActionEnabled(context.Background(), testCase.ActionName, testCase.Enabled)
			assert.Equal(t, testCase.Err, err)

			actions, err := manager.ListActions(context.Background())
			assert.NoError(t, err)
			assert.Equal(t, testCase.Expected, actions)
		})
	}
}
