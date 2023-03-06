package reloader

import (
	"testing"
	"time"

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

		NewConfig config.Config
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
			NewConfig:       config.Config{},
			NewResVer:       2,
			ExpectedCfg:     config.Config{},
			ExpectedResVer:  2,
			ExpectedCfgDiff: configDiff{},
		},
		{
			Name:          "Resource Version lower",
			InitialCfg:    config.Config{},
			InitialResVer: 2,
			NewConfig:     config.Config{},
			NewResVer:     1,

			ExpectedErrMessage: "current config version (2) is newer than the latest one (1)",
			ExpectedCfg:        config.Config{},
			ExpectedResVer:     2,
			ExpectedCfgDiff:    configDiff{},
		},
		{
			Name: "Resource Version higher but config the same",
			InitialCfg: config.Config{
				Actions: map[string]config.Action{
					"test": {
						Enabled: false,
					},
				},
			},
			InitialResVer: 2,
			NewConfig: config.Config{
				Actions: map[string]config.Action{
					"test": {
						Enabled: false,
					},
				},
			},
			NewResVer: 3,

			ExpectedErrMessage: "",
			ExpectedCfg: config.Config{
				Actions: map[string]config.Action{
					"test": {
						Enabled: false,
					},
				},
			},
			ExpectedResVer:  3,
			ExpectedCfgDiff: configDiff{},
		},
		{
			Name: "Different config, should restart",
			InitialCfg: config.Config{
				Actions: map[string]config.Action{
					"test": {
						Enabled: false,
					},
					"second": {
						Enabled: false,
					},
				},
			},
			InitialResVer: 2,
			NewConfig: config.Config{
				Actions: map[string]config.Action{
					"test": {
						Enabled: true,
					},
					"second": {
						Enabled: false,
					},
				},
			},
			NewResVer: 3,

			ExpectedErrMessage: "",
			ExpectedCfg: config.Config{
				Actions: map[string]config.Action{
					"test": {
						Enabled: true,
					},
					"second": {
						Enabled: false,
					},
				},
			},
			ExpectedResVer:  3,
			ExpectedCfgDiff: configDiff{shouldRestart: true},
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
				latestCfg:     testCase.InitialCfg,
				resVersion:    testCase.InitialResVer,
			}

			cfgDiff, err := remoteReloader.processNewConfig(testCase.NewConfig, testCase.NewResVer)
			if testCase.ExpectedErrMessage != "" {
				require.Error(t, err)
				assert.EqualError(t, err, testCase.ExpectedErrMessage)
				assert.Equal(t, testCase.ExpectedCfg, remoteReloader.latestCfg)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, testCase.ExpectedCfg, remoteReloader.latestCfg)
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
