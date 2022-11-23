package plugin

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/hashicorp/go-plugin"
	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/api/executor"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/multierror"
)

// ErrNotStartedPluginManager is an error returned when Plugin Manager was not yet started and initialized successfully.
var ErrNotStartedPluginManager = errors.New("plugin manager is not started yet")

const (
	executorPluginName = "executor"
	dirPerms           = 0o775
	binPerms           = 0o755
	filePerms          = 0o664
)

// pluginMap is the map of plugins we can dispense.
// This map is used in order to identify a plugin called Dispense.
// This map is globally available and must stay consistent in order for all the plugins to work.
var pluginMap = map[string]plugin.Plugin{
	// TODO(plugin-sources): add me:
	//sourcePluginName:   &source.Plugin{},
	executorPluginName: &executor.Plugin{},
}

// Manager provides functionality for managing executor and source plugins.
type Manager struct {
	isStarted  atomic.Bool
	log        logrus.FieldLogger
	cfg        config.Plugins
	httpClient *http.Client

	executorsToEnable []string
	executorsStore    store[executor.Executor]
}

// NewManager returns a new Manager instance.
func NewManager(logger logrus.FieldLogger, cfg config.Plugins, executors []string) *Manager {
	return &Manager{
		cfg:               cfg,
		httpClient:        newHTTPClient(),
		executorsToEnable: executors,
		executorsStore:    newStore[executor.Executor](),
		log:               logger.WithField("component", "Plugin Manager"),
	}
}

// Start downloads and starts all enabled plugins.
func (m *Manager) Start(ctx context.Context) error {
	if len(m.executorsToEnable) == 0 {
		m.log.Info("No external plugins are enabled.")
		return nil
	}

	m.log.WithFields(logrus.Fields{
		"enabledExecutors": strings.Join(m.executorsToEnable, ","),
	}).Info("Starting Plugin Manager for all enabled plugins")

	if err := m.loadRepositoriesMetadata(ctx); err != nil {
		return err
	}

	executorPlugins, err := m.loadPlugins(ctx, executorPluginName, m.executorsToEnable, m.executorsStore.Repository)
	if err != nil {
		return err
	}
	executorClients, err := createGRPCClients[executor.Executor](executorPlugins, executorPluginName)
	if err != nil {
		return fmt.Errorf("while creating executor plugins: %w", err)
	}
	m.executorsStore.EnabledPlugins = executorClients

	m.isStarted.Store(true)
	return nil
}

// GetExecutor returns the executor client for a given plugin.
func (m *Manager) GetExecutor(name string) (executor.Executor, error) {
	if !m.isStarted.Load() {
		return nil, ErrNotStartedPluginManager
	}

	client, found := m.executorsStore.EnabledPlugins[name]
	if !found || client.Client == nil {
		return nil, fmt.Errorf("client for plugin %q not found", name)
	}

	return client.Client, nil
}

// Shutdown performs any necessary cleanup.
// This method blocks until all cleanup is finished.
func (m *Manager) Shutdown() {
	var wg sync.WaitGroup
	for _, p := range m.executorsStore.EnabledPlugins {
		wg.Add(1)

		go func(close func()) {
			if close != nil {
				close()
			}
			wg.Done()
		}(p.Cleanup)
	}
	wg.Wait()
}

func (m *Manager) loadPlugins(ctx context.Context, pluginType string, pluginsToEnable []string, repo map[string][]IndexEntry) (map[string]string, error) {
	loadedPlugins := map[string]string{}
	for _, pluginKey := range pluginsToEnable {
		candidates, found := repo[pluginKey]
		if !found || len(candidates) == 0 {
			return nil, fmt.Errorf("not found %q plugin in any repository", pluginKey)
		}
		// TODO(version): check if version is defined in plugin:
		// - if yes, use it.
		// - if not, find the latest version in the repository.
		pluginInfo := candidates[0]

		repoName, pluginName, err := DecomposePluginKey(pluginKey)
		if err != nil {
			return nil, err
		}
		binPath := filepath.Join(m.cfg.CacheDir, repoName, fmt.Sprintf("%s_%s_%s", pluginType, pluginInfo.Version, pluginName))

		log := m.log.WithFields(logrus.Fields{
			"plugin":  pluginKey,
			"version": pluginInfo.Version,
			"binPath": binPath,
		})

		if _, err := os.Stat(binPath); os.IsNotExist(err) {
			log.Debug("Executor plugin not found locally")
			err := m.downloadPlugin(ctx, binPath, pluginInfo)
			if err != nil {
				return nil, fmt.Errorf("while fetching plugin %q binary: %w", pluginKey, err)
			}
		}

		loadedPlugins[pluginKey] = binPath

		log.Info("Executor plugin registered successfully.")
	}

	return loadedPlugins, nil
}

func (m *Manager) loadRepositoriesMetadata(ctx context.Context) error {
	// index repositories metadata
	for repo, url := range m.cfg.Repositories {
		path := filepath.Join(m.cfg.CacheDir, filepath.Clean(fmt.Sprintf("%s.yaml", repo)))
		if _, err := os.Stat(path); os.IsNotExist(err) {
			err := m.fetchIndex(ctx, path, url)
			if err != nil {
				return fmt.Errorf("while fetching index for %q repository: %w", repo, err)
			}
		}

		data, err := os.ReadFile(filepath.Clean(path))
		if err != nil {
			return fmt.Errorf("while reading index file: %w", err)
		}
		var index Index
		if err := yaml.Unmarshal(data, &index); err != nil {
			return fmt.Errorf("while unmarshaling index: %w", err)
		}

		for _, entry := range index.Entries {
			key := BuildPluginKey(repo, entry.Name)
			switch entry.Type {
			case TypeExecutor:
				m.executorsStore.Repository[key] = append(m.executorsStore.Repository[key], entry)
			case TypeSource:
				// TODO(plugin-sources): add me.
			}
		}
	}

	// sort entries by version
	for key := range m.executorsStore.Repository {
		sort.Sort(byIndexEntryVersion(m.executorsStore.Repository[key]))
	}
	// TODO(plugin-sources): add sorting

	return nil
}

func (m *Manager) fetchIndex(ctx context.Context, path, url string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("while creating request: %w", err)
	}

	res, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("while executing request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("incorrect status code: %d", res.StatusCode)
	}

	err = os.MkdirAll(filepath.Dir(path), dirPerms)
	if err != nil {
		return fmt.Errorf("while creating directory where repository index should be stored: %w", err)
	}
	file, err := os.OpenFile(filepath.Clean(path), os.O_CREATE|os.O_WRONLY, filePerms)
	if err != nil {
		return fmt.Errorf("while creating file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, res.Body)
	if err != nil {
		return fmt.Errorf("while saving index body: %w", err)
	}
	return nil
}

func createGRPCClients[C any](bins map[string]string, dispenseType string) (map[string]enabledPlugins[C], error) {
	out := map[string]enabledPlugins[C]{}

	for key, path := range bins {
		cli := plugin.NewClient(&plugin.ClientConfig{
			Plugins: pluginMap,
			//nolint:gosec // warns us about 'Subprocess launching with variable', but we are the one that created that variable.
			Cmd:              exec.Command(path),
			AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
			HandshakeConfig: plugin.HandshakeConfig{
				ProtocolVersion:  executor.ProtocolVersion,
				MagicCookieKey:   api.HandshakeConfig.MagicCookieKey,
				MagicCookieValue: api.HandshakeConfig.MagicCookieValue,
			},
		})

		rpcClient, err := cli.Client()
		if err != nil {
			return nil, err
		}

		raw, err := rpcClient.Dispense(dispenseType)
		if err != nil {
			return nil, err
		}

		concreteCli, ok := raw.(C)
		if !ok {
			cli.Kill()
			return nil, fmt.Errorf("registered client doesn't implemented executor interface")
		}
		out[key] = enabledPlugins[C]{
			Client:  concreteCli,
			Cleanup: cli.Kill,
		}
	}

	return out, nil
}

func (m *Manager) downloadPlugin(ctx context.Context, binPath string, info IndexEntry) error {
	err := os.MkdirAll(filepath.Dir(binPath), dirPerms)
	if err != nil {
		return fmt.Errorf("while creating directory where plugin should be stored: %w", err)
	}

	suffix := fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH)
	getDownloadURL := func() string {
		for _, url := range info.Links {
			if strings.HasSuffix(url, suffix) {
				return url
			}
		}
		return ""
	}

	url := getDownloadURL()
	if url == "" {
		return fmt.Errorf("cannot find download url with suffix %s", suffix)
	}

	m.log.WithFields(logrus.Fields{
		"url":     url,
		"binPath": binPath,
	}).Info("Downloading plugin.")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return fmt.Errorf("while creating request: %w", err)
	}

	res, err := m.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("while executing request: %w", err)
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("incorrect status code: %d", res.StatusCode)
	}

	file, err := os.OpenFile(filepath.Clean(binPath), os.O_RDWR|os.O_CREATE|os.O_TRUNC, binPerms)
	if err != nil {
		return fmt.Errorf("while creating plugin file: %w", err)
	}

	_, err = io.Copy(file, res.Body)
	file.Close()
	if err != nil {
		err := multierror.Append(err, os.Remove(binPath))
		return fmt.Errorf("while downloading file: %w", err.ErrorOrNil())
	}

	return nil
}
