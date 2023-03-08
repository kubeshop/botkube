package reloader

import (
	"testing"
	"time"

	"github.com/MakeNowJust/heredoc"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/kubeshop/botkube/internal/loggerx"
	"github.com/kubeshop/botkube/pkg/config"
)

func TestRemote_ProcessConfig(t *testing.T) {
	testCases := []struct {
		Name string

		InitialCfg    config.Config
		InitialResVer int

		NewConfig string
		NewResVer int

		ExpectedErrMessage string
		ExpectedCfg        config.Config
		ExpectedCfgDiff    configDiff
		ExpectedResVer     int
	}{
		{
			Name:            "Resource Version equal",
			InitialCfg:      config.Config{},
			InitialResVer:   2,
			NewConfig:       "",
			NewResVer:       2,
			ExpectedCfg:     config.Config{},
			ExpectedResVer:  2,
			ExpectedCfgDiff: configDiff{},
		},
		{
			Name:          "Resource Version lower",
			InitialCfg:    config.Config{},
			InitialResVer: 2,
			NewConfig:     "",
			NewResVer:     1,

			ExpectedErrMessage: "while comparing config versions: current config version (2) is newer than the latest one (1)",
			ExpectedCfg:        config.Config{},
			ExpectedResVer:     2,
			ExpectedCfgDiff:    configDiff{},
		},
		{
			Name:          "Resource Version higher but config the same",
			InitialCfg:    fixConfig(true),
			InitialResVer: 2,
			NewConfig:     fixConfigStr(true),
			NewResVer:     3,

			ExpectedErrMessage: "",
			ExpectedCfg:        fixConfig(true),
			ExpectedResVer:     3,
			ExpectedCfgDiff:    configDiff{},
		},
		{
			Name:          "Different config, should restart",
			InitialCfg:    fixConfig(false),
			InitialResVer: 2,
			NewConfig:     fixConfigStr(true),
			NewResVer:     3,

			ExpectedErrMessage: "",
			ExpectedCfg:        fixConfig(true),
			ExpectedResVer:     3,
			ExpectedCfgDiff:    configDiff{shouldRestart: true},
		},
	}

	for _, testCase := range testCases {
		t.Run(testCase.Name, func(t *testing.T) {
			resVerHldr := &sampleResVerHolder{testCase.InitialResVer}
			remoteReloader := RemoteConfigReloader{
				log:           loggerx.NewNoop(),
				interval:      time.Minute,
				deployCli:     nil,
				resVerHolders: []ResourceVersionHolder{resVerHldr},
				currentCfg:    testCase.InitialCfg,
				resVersion:    testCase.InitialResVer,
			}

			cfgDiff, err := remoteReloader.processNewConfig([]byte(testCase.NewConfig), testCase.NewResVer)
			if testCase.ExpectedErrMessage != "" {
				require.Error(t, err)
				assert.EqualError(t, err, testCase.ExpectedErrMessage)
				assert.Equal(t, testCase.ExpectedCfg, remoteReloader.currentCfg)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, testCase.ExpectedCfg, remoteReloader.currentCfg)
			assert.Equal(t, testCase.ExpectedResVer, remoteReloader.resVersion)
			assert.Equal(t, testCase.ExpectedResVer, resVerHldr.resVer)
			assert.Equal(t, testCase.ExpectedCfgDiff, cfgDiff)
		})
	}
}

type sampleResVerHolder struct {
	resVer int
}

func (s *sampleResVerHolder) SetResourceVersion(version int) {
	s.resVer = version
}

func (s *sampleResVerHolder) GetResourceVersion(version int) {
	s.resVer = version
}

func fixConfig(actionEnabled bool) config.Config {
	return config.Config{
		Communications: map[string]config.Communications{
			"test": {
				SocketSlack: config.SocketSlack{
					Enabled:  true,
					AppToken: "xapp-123",
					BotToken: "xoxb-123",
					Channels: map[string]config.ChannelBindingsByName{
						"foo": {
							Notification: config.ChannelNotification{
								Disabled: false,
							},
						},
					},
				},
			},
		},
		Actions: map[string]config.Action{
			"test": {
				Enabled: actionEnabled,
				Command: "test",
			},
		},
		// defaults from the /pkg/config/default.yaml file
		Settings: config.Settings{
			MetricsPort: "2112",
			HealthPort:  "2114",
			Log: config.Logger{
				Level: "error",
			},
			InformersResyncPeriod: 30 * time.Minute,
			SystemConfigMap: config.K8sResourceRef{
				Name:      "botkube-system",
				Namespace: "botkube",
			},
		},
		Plugins: config.PluginManagement{
			CacheDir: "/tmp",
		},
		ConfigWatcher: config.CfgWatcher{
			Remote: config.RemoteCfgWatcher{
				PollInterval: 15 * time.Second,
			},
		},
	}
}

func fixConfigStr(actionEnabled bool) string {
	return heredoc.Docf(`
		communications:
		  test:
		    socketSlack:
		      enabled: true
		      appToken: xapp-123
		      botToken: xoxb-123
		      channels:
		        foo:
		          notification:
		            disabled: false
		actions:
		  test:	
		    enabled: %v
		    command: "test"
		`, actionEnabled)
}
