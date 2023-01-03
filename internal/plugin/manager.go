package plugin

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/hashicorp/go-plugin"
	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/api/executor"
	"github.com/kubeshop/botkube/pkg/api/source"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/multierror"
)

const (
	dirPerms  = 0o775
	binPerms  = 0o755
	filePerms = 0o664
)

// pluginMap is the map of plugins we can dispense.
// This map is used in order to identify a plugin called Dispense.
// This map is globally available and must stay consistent in order for all the plugins to work.
var pluginMap = map[string]plugin.Plugin{
	TypeSource.String():   &source.Plugin{},
	TypeExecutor.String(): &executor.Plugin{},
}

// Manager provides functionality for managing executor and source plugins.
type Manager struct {
	isStarted  atomic.Bool
	log        logrus.FieldLogger
	cfg        config.Plugins
	httpClient *http.Client

	executorsToEnable []string
	executorsStore    store[executor.Executor]

	sourcesStore    store[source.Source]
	sourcesToEnable []string
}

// NewManager returns a new Manager instance.
func NewManager(logger logrus.FieldLogger, cfg config.Plugins, executors, sources []string) *Manager {
	return &Manager{
		cfg:               cfg,
		httpClient:        newHTTPClient(),
		executorsToEnable: executors,
		executorsStore:    newStore[executor.Executor](),
		sourcesToEnable:   sources,
		sourcesStore:      newStore[source.Source](),
		log:               logger.WithField("component", "Plugin Manager"),
	}
}

// Start downloads and starts all enabled plugins.
func (m *Manager) Start(ctx context.Context) error {
	if len(m.executorsToEnable) == 0 && len(m.sourcesToEnable) == 0 {
		m.log.Info("No external plugins are enabled.")
		return nil
	}

	m.log.WithFields(logrus.Fields{
		"enabledExecutors": strings.Join(m.executorsToEnable, ","),
		"enabledSources":   strings.Join(m.sourcesToEnable, ","),
	}).Info("Starting Plugin Manager for all enabled plugins")

	err := m.start(ctx, false)
	switch {
	case err == nil:
	case IsNotFoundError(err):
		m.log.Infof("%s. Retrying Plugin Manager start with forced repo index update.", err)
		return m.start(ctx, true)
	default:
		return err
	}

	m.isStarted.Store(true)
	return nil
}

func (m *Manager) start(ctx context.Context, forceUpdate bool) error {
	if err := m.loadRepositoriesMetadata(ctx, forceUpdate); err != nil {
		return err
	}

	executorPlugins, err := m.loadPlugins(ctx, TypeExecutor, m.executorsToEnable, m.executorsStore.Repository)
	if err != nil {
		return err
	}

	executorClients, err := createGRPCClients[executor.Executor](m.log, executorPlugins, TypeExecutor)
	if err != nil {
		return fmt.Errorf("while creating executor plugins: %w", err)
	}
	m.executorsStore.EnabledPlugins = executorClients

	sourcesPlugins, err := m.loadPlugins(ctx, TypeSource, m.sourcesToEnable, m.sourcesStore.Repository)
	if err != nil {
		return err
	}
	sourcesClients, err := createGRPCClients[source.Source](m.log, sourcesPlugins, TypeSource)
	if err != nil {
		return fmt.Errorf("while creating source plugins: %w", err)
	}
	m.sourcesStore.EnabledPlugins = sourcesClients

	return nil
}

// GetExecutor returns the executor client for a given plugin.
func (m *Manager) GetExecutor(name string) (executor.Executor, error) {
	if !m.isStarted.Load() {
		return nil, ErrNotStartedPluginManager
	}

	client, found := m.executorsStore.EnabledPlugins[name]
	if !found || client.Client == nil {
		return nil, fmt.Errorf("client for executor plugin %q not found", name)
	}

	return client.Client, nil
}

// GetSource returns the source client for a given plugin.
func (m *Manager) GetSource(name string) (source.Source, error) {
	if !m.isStarted.Load() {
		return nil, ErrNotStartedPluginManager
	}

	client, found := m.sourcesStore.EnabledPlugins[name]
	if !found || client.Client == nil {
		return nil, fmt.Errorf("client for source plugin %q not found", name)
	}

	return client.Client, nil
}

// Shutdown performs any necessary cleanup.
// This method blocks until all cleanup is finished.
func (m *Manager) Shutdown() {
	var wg sync.WaitGroup
	releasePlugins(&wg, m.sourcesStore.EnabledPlugins)
	releasePlugins(&wg, m.executorsStore.EnabledPlugins)
	wg.Wait()
}

func releasePlugins[T any](wg *sync.WaitGroup, enabledPlugins storePlugins[T]) {
	for _, p := range enabledPlugins {
		wg.Add(1)

		go func(close func()) {
			if close != nil {
				close()
			}
			wg.Done()
		}(p.Cleanup)
	}
}

func (m *Manager) loadPlugins(ctx context.Context, pluginType Type, pluginsToEnable []string, repo storeRepository) (map[string]string, error) {
	loadedPlugins := map[string]string{}
	for _, pluginKey := range pluginsToEnable {
		repoName, pluginName, ver, err := config.DecomposePluginKey(pluginKey)
		if err != nil {
			return nil, err
		}

		candidates, found := repo.Get(repoName, pluginName)
		if !found || len(candidates) == 0 {
			return nil, NewNotFoundPluginError("not found %s plugin called %q in %q repository", pluginType.String(), pluginName, repoName)
		}

		// entries are sorted by version, first is the latest one.
		latestPluginInfo := candidates[0]

		// if plugin version not defined by user, use the latest one
		if ver == "" {
			ver = latestPluginInfo.Version
		}

		binPath := filepath.Join(m.cfg.CacheDir, repoName, fmt.Sprintf("%s_%s_%s", pluginType, ver, pluginName))

		log := m.log.WithFields(logrus.Fields{
			"plugin":  pluginKey,
			"version": ver,
			"binPath": binPath,
		})

		if _, err := os.Stat(binPath); os.IsNotExist(err) {
			log.Debug("Executor plugin not found locally")
			err := m.downloadPlugin(ctx, binPath, latestPluginInfo)
			if err != nil {
				return nil, fmt.Errorf("while fetching plugin %q binary: %w", pluginKey, err)
			}
		}

		loadedPlugins[pluginKey] = binPath

		log.Info("Executor plugin registered successfully.")
	}

	return loadedPlugins, nil
}

func (m *Manager) loadRepositoriesMetadata(ctx context.Context, forceUpdate bool) error {
	rawIndexes := map[string][]byte{}
	for repo, entry := range m.cfg.Repositories {
		path := filepath.Join(m.cfg.CacheDir, filepath.Clean(fmt.Sprintf("%s.yaml", repo)))

		if _, err := os.Stat(path); forceUpdate || os.IsNotExist(err) {
			m.log.WithFields(logrus.Fields{
				"repo":        repo,
				"url":         entry.URL,
				"forceUpdate": forceUpdate,
			}).Debug("Downloading repository index")

			err := m.fetchIndex(ctx, path, entry.URL)
			if err != nil {
				return fmt.Errorf("while fetching index for %q repository: %w", repo, err)
			}
		}

		data, err := os.ReadFile(filepath.Clean(path))
		if err != nil {
			return fmt.Errorf("while reading index file: %w", err)
		}

		rawIndexes[repo] = data
	}

	executorsRepos, sourcesRepos, err := newStoreRepositories(rawIndexes)
	if err != nil {
		return fmt.Errorf("while building repositories store: %w", err)
	}
	m.executorsStore.Repository = executorsRepos
	m.sourcesStore.Repository = sourcesRepos

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

func createGRPCClients[C any](logger logrus.FieldLogger, bins map[string]string, pluginType Type) (map[string]enabledPlugins[C], error) {
	out := map[string]enabledPlugins[C]{}

	for key, path := range bins {
		pluginLogger, stdoutLogger, stderrLogger := NewPluginLoggers(logger, key, pluginType)

		cli := plugin.NewClient(&plugin.ClientConfig{
			Plugins: pluginMap,
			//nolint:gosec // warns us about 'Subprocess launching with variable', but we are the one that created that variable.
			Cmd:              newPluginOSRunCommand(path),
			AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
			HandshakeConfig: plugin.HandshakeConfig{
				ProtocolVersion:  executor.ProtocolVersion,
				MagicCookieKey:   api.HandshakeConfig.MagicCookieKey,
				MagicCookieValue: api.HandshakeConfig.MagicCookieValue,
			},

			Logger:     pluginLogger,
			SyncStdout: stdoutLogger,
			SyncStderr: stderrLogger,
		})

		rpcClient, err := cli.Client()
		if err != nil {
			return nil, err
		}

		raw, err := rpcClient.Dispense(pluginType.String())
		if err != nil {
			return nil, err
		}

		concreteCli, ok := raw.(C)
		if !ok {
			cli.Kill()
			return nil, fmt.Errorf("registered client doesn't implement required %s interface", pluginType.String())
		}

		out[key] = enabledPlugins[C]{
			Client:  concreteCli,
			Cleanup: cli.Kill,
		}
	}

	return out, nil
}

func newPluginOSRunCommand(path string) *exec.Cmd {
	cmd := exec.Command(path)
	val, found := os.LookupEnv("KUBECONFIG")
	if found {
		cmd.Env = []string{fmt.Sprintf("KUBECONFIG=%s", val)}
	}
	return cmd
}

func (m *Manager) downloadPlugin(ctx context.Context, binPath string, info storeEntry) error {
	err := os.MkdirAll(filepath.Dir(binPath), dirPerms)
	if err != nil {
		return fmt.Errorf("while creating directory where plugin should be stored: %w", err)
	}

	selector := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)

	url, found := info.URLs[selector]
	if !found {
		return NewNotFoundPluginError("cannot find download url for %s", selector)
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
