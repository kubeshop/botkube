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
	"time"

	"github.com/hashicorp/go-plugin"
	"github.com/sirupsen/logrus"
	stringutil "k8s.io/utils/strings"

	"github.com/kubeshop/botkube/internal/config/remote"
	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/api/executor"
	"github.com/kubeshop/botkube/pkg/api/source"
	"github.com/kubeshop/botkube/pkg/config"
	"github.com/kubeshop/botkube/pkg/formatx"
	"github.com/kubeshop/botkube/pkg/httpx"
	"github.com/kubeshop/botkube/pkg/multierror"
	"github.com/kubeshop/botkube/pkg/templatex"
)

const (
	dirPerms  = 0o775
	binPerms  = 0o755
	filePerms = 0o664

	// DependencyDirEnvName define environment variable where plugin dependency binaries are stored.
	DependencyDirEnvName = "PLUGIN_DEPENDENCY_DIR"

	defaultHealthCheckInterval = 10 * time.Second
	printHeaderValueCharCount  = 3
)

// pluginMap is the map of plugins we can dispense.
// This map is used in order to identify a plugin called Dispense.
// This map is globally available and must stay consistent in order for all the plugins to work.
var pluginMap = map[string]plugin.Plugin{
	TypeSource.String():   &source.Plugin{},
	TypeExecutor.String(): &executor.Plugin{},
}

// IndexRenderData returns plugin index render data.
type IndexRenderData struct {
	Remote remote.Config `yaml:"remote"`
}

// Manager provides functionality for managing executor and source plugins.
type Manager struct {
	isStarted       atomic.Bool
	log             logrus.FieldLogger
	logConfig       config.Logger
	cfg             config.PluginManagement
	httpClient      *http.Client
	indexRenderData IndexRenderData

	sourceSupervisorChan   chan pluginMetadata
	executorSupervisorChan chan pluginMetadata
	schedulerChan          chan string

	executorsToEnable []string
	executorsStore    *store[executor.Executor]

	sourcesStore    *store[source.Source]
	sourcesToEnable []string

	healthCheckInterval time.Duration
	monitor             *HealthMonitor
}

type pluginMetadata struct {
	binPath   string
	pluginKey string
}

// NewManager returns a new Manager instance.
func NewManager(logger logrus.FieldLogger, logCfg config.Logger, cfg config.PluginManagement, executors, sources []string, schedulerChan chan string, stats *HealthStats) *Manager {
	sourceSupervisorChan := make(chan pluginMetadata)
	executorSupervisorChan := make(chan pluginMetadata)
	executorsStore := newStore[executor.Executor]()
	sourcesStore := newStore[source.Source]()

	remoteCfg, _ := remote.GetConfig()
	indexRenderData := IndexRenderData{
		Remote: remoteCfg,
	}

	return &Manager{
		cfg:                    cfg,
		httpClient:             httpx.NewHTTPClient(),
		indexRenderData:        indexRenderData,
		sourceSupervisorChan:   sourceSupervisorChan,
		executorSupervisorChan: executorSupervisorChan,
		schedulerChan:          schedulerChan,
		executorsToEnable:      executors,
		executorsStore:         &executorsStore,
		sourcesToEnable:        sources,
		sourcesStore:           &sourcesStore,
		log:                    logger.WithField("component", "Plugin Manager"),
		logConfig:              logCfg, // used when we create on-demand loggers for plugins
		healthCheckInterval:    cfg.HealthCheckInterval,
		monitor: NewHealthMonitor(
			logger.WithField("component", "Plugin Health Monitor"),
			logCfg,
			cfg.RestartPolicy,
			schedulerChan,
			sourceSupervisorChan,
			executorSupervisorChan,
			&executorsStore,
			&sourcesStore,
			cfg.HealthCheckInterval,
			stats,
		),
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

	m.monitor.Start(ctx)

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

	executorClients, err := createGRPCClients[executor.Executor](ctx, m.log, m.logConfig, executorPlugins, TypeExecutor, m.executorSupervisorChan, m.healthCheckInterval)
	if err != nil {
		return fmt.Errorf("while creating executor plugins: %w", err)
	}
	m.executorsStore.EnabledPlugins = executorClients

	sourcesPlugins, err := m.loadPlugins(ctx, TypeSource, m.sourcesToEnable, m.sourcesStore.Repository)
	if err != nil {
		return err
	}
	sourcesClients, err := createGRPCClients[source.Source](ctx, m.log, m.logConfig, sourcesPlugins, TypeSource, m.sourceSupervisorChan, m.healthCheckInterval)
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

	client, found := m.executorsStore.EnabledPlugins.Get(name)
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

	client, found := m.sourcesStore.EnabledPlugins.Get(name)
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

func releasePlugins[T any](wg *sync.WaitGroup, enabledPlugins *storePlugins[T]) {
	for _, p := range enabledPlugins.data {
		wg.Add(1)

		go func(close func()) {
			if close != nil {
				close()
			}
			wg.Done()
		}(p.Cleanup)
	}
}

func (m *Manager) loadPlugins(ctx context.Context, pluginType Type, pluginsToEnable []string, repo storeRepository) (map[string]pluginMetadata, error) {
	loadedPlugins := map[string]pluginMetadata{}
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

		err = m.ensurePluginDownloaded(ctx, binPath, latestPluginInfo)
		if err != nil {
			return nil, fmt.Errorf("while fetching plugin %q binary: %w", pluginKey, err)
		}

		loadedPlugins[pluginKey] = pluginMetadata{
			pluginKey: pluginKey,
			binPath:   binPath,
		}

		log.Infof("%s plugin registered successfully.", formatx.ToTitle(pluginType))
	}

	return loadedPlugins, nil
}

func (m *Manager) collectEnabledRepositories() ([]string, error) {
	issues := multierror.New()

	collect := func(in []string, pType Type) []string {
		var out []string
		for _, pluginKey := range in {
			repoName, _, _, err := config.DecomposePluginKey(pluginKey)
			if err != nil {
				issues = multierror.Append(issues, err)
				continue
			}

			_, found := m.cfg.Repositories[repoName]
			if !found {
				issues = multierror.Append(issues, fmt.Errorf("repository %q is not defined, but it is referred by %s plugin called %q", repoName, pType, pluginKey))
				continue
			}

			out = append(out, repoName)
		}
		return out
	}

	requestedRepositories := collect(m.executorsToEnable, TypeExecutor)
	requestedRepositories = append(requestedRepositories, collect(m.sourcesToEnable, TypeSource)...)

	if err := issues.ErrorOrNil(); err != nil {
		return nil, err
	}

	return requestedRepositories, nil
}

func (m *Manager) loadRepositoriesMetadata(ctx context.Context, forceUpdate bool) error {
	repos, err := m.collectEnabledRepositories()
	if err != nil {
		return err
	}

	rawIndexes := map[string][]byte{}
	for _, repo := range repos {
		entry := m.cfg.Repositories[repo]
		path := filepath.Join(m.cfg.CacheDir, filepath.Clean(fmt.Sprintf("%s.yaml", repo)))

		if _, err := os.Stat(path); forceUpdate || os.IsNotExist(err) {
			m.log.WithFields(logrus.Fields{
				"repo":        repo,
				"url":         entry.URL,
				"forceUpdate": forceUpdate,
			}).Info("Downloading repository index")

			err := m.fetchIndex(ctx, path, entry)
			if err != nil {
				return fmt.Errorf("while fetching index for %q repository with URL %q: %w", repo, entry.URL, err)
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

func (m *Manager) fetchIndex(ctx context.Context, path string, repo config.PluginsRepository) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, repo.URL, http.NoBody)
	if err != nil {
		return fmt.Errorf("while creating request: %w", err)
	}

	headers, err := m.renderPluginIndexHeaders(repo.Headers)
	if err != nil {
		return fmt.Errorf("while rendering plugin index header: %w", err)
	}

	var strBuilder strings.Builder
	for key, value := range headers {
		strBuilder.WriteString(fmt.Sprintf("%s=%s\n", key, stringutil.ShortenString(value, printHeaderValueCharCount)))
		req.Header.Set(key, value)
	}

	m.log.WithFields(logrus.Fields{
		"headers": strBuilder.String(),
		"url":     repo.URL,
	}).Debug("Fetching index via GET request...")

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

func (m *Manager) renderPluginIndexHeaders(headers map[string]string) (map[string]string, error) {
	out := make(map[string]string)

	errs := multierror.New()
	for key, value := range headers {
		renderedValue, err := templatex.RenderStringIfTemplate(value, m.indexRenderData)
		if err != nil {
			errs = multierror.Append(errs, fmt.Errorf("while rendering header %q: %w", key, err))
			continue
		}

		out[key] = renderedValue
	}

	return out, errs.ErrorOrNil()
}

func createGRPCClients[C any](ctx context.Context, logger logrus.FieldLogger, logConfig config.Logger, pluginMeta map[string]pluginMetadata, pluginType Type, supervisorChan chan pluginMetadata, healthCheckInterval time.Duration) (*storePlugins[C], error) {
	out := map[string]enabledPlugins[C]{}
	for key, pm := range pluginMeta {
		p, err := createGRPCClient[C](ctx, logger, logConfig, pm, pluginType, supervisorChan, healthCheckInterval)
		if err != nil {
			return nil, fmt.Errorf("while creating GRPC client for %s plugin %q: %w", pluginType.String(), key, err)
		}
		out[key] = p
	}

	return &storePlugins[C]{data: out}, nil
}

func createGRPCClient[C any](ctx context.Context, logger logrus.FieldLogger, logConfig config.Logger, pm pluginMetadata, pluginType Type, supervisorChan chan pluginMetadata, healthCheckInterval time.Duration) (enabledPlugins[C], error) {
	pluginLogger, stdoutLogger, stderrLogger := NewPluginLoggers(logger, logConfig, pm.pluginKey, pluginType)

	cli := plugin.NewClient(&plugin.ClientConfig{
		Plugins: pluginMap,
		//nolint:gosec // warns us about 'Subprocess launching with variable', but we are the one that created that variable.
		Cmd:              newPluginOSRunCommand(pm.binPath),
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
		return enabledPlugins[C]{}, err
	}

	raw, err := rpcClient.Dispense(pluginType.String())
	if err != nil {
		return enabledPlugins[C]{}, err
	}

	concreteCli, ok := raw.(C)
	if !ok {
		cli.Kill()
		return enabledPlugins[C]{}, fmt.Errorf("registered client doesn't implement required %s interface", pluginType.String())
	}

	startPluginHealthWatcher(ctx, logger, rpcClient, pm, supervisorChan, healthCheckInterval)

	return enabledPlugins[C]{
		Client:  concreteCli,
		Cleanup: cli.Kill,
	}, nil
}

func startPluginHealthWatcher(ctx context.Context, logger logrus.FieldLogger, rpcClient plugin.ClientProtocol, pm pluginMetadata, supervisorChan chan pluginMetadata, healthCheckInterval time.Duration) {
	logger.Infof("Starting plugin %q health watcher...", pm.pluginKey)
	interval := healthCheckInterval
	if interval.Seconds() < 1 {
		interval = defaultHealthCheckInterval
	}
	ticker := time.NewTicker(interval)
	go func() {
		for {
			select {
			case <-ticker.C:
				if err := rpcClient.Ping(); err != nil {
					logger.WithError(err).Errorf("Plugin %q is not responding.", pm.pluginKey)
					logger.WithField("name", pm.pluginKey).Debugf("Informing supervisor to restart plugin...")
					supervisorChan <- pm
					return
				}

				logger.Debugf("Plugin %q is responding.", pm.pluginKey)

			case <-ctx.Done():
				logger.Infof("Exiting plugin %q supervisor...", pm.pluginKey)
				return
			}
		}
	}()
}

func newPluginOSRunCommand(path string) *exec.Cmd {
	cmd := exec.Command(path)

	// Set env with path to dependencies
	//
	// Unfortunately, we cannot override PATH env variable when creating a plugin client.
	// The `go-plugin` calls os.Environ() and, in a result, overrides modified envs passed to the plugin client.
	// See: https://github.com/hashicorp/go-plugin/blob/9d19a83630e51cd9e141c140fb0d8384818849de/client.go#L554-L556
	// So the only way is to use a custom env variable which won't be overridden by the os.Environ() call in the main process.
	depDir := dependencyDirForBin(path)
	cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", DependencyDirEnvName, depDir))

	// Set Kubeconfig env
	val, found := os.LookupEnv("KUBECONFIG")
	if found {
		cmd.Env = append(cmd.Env, fmt.Sprintf("KUBECONFIG=%s", val))
	}

	return cmd
}

func (m *Manager) ensurePluginDownloaded(ctx context.Context, binPath string, info storeEntry) error {
	selector := fmt.Sprintf("%s/%s", runtime.GOOS, runtime.GOARCH)

	log := m.log.WithFields(logrus.Fields{
		"binPath": binPath,
	})

	// Ensure plugin downloaded
	if !DoesBinaryExist(binPath) {
		err := os.MkdirAll(filepath.Dir(binPath), dirPerms)
		if err != nil {
			return fmt.Errorf("while creating directory where plugin should be stored: %w", err)
		}

		url, found := info.URLs[selector]
		if !found {
			return NewNotFoundPluginError("cannot find download url for %s", selector)
		}

		log.WithFields(logrus.Fields{
			"url": url,
		}).Info("Downloading plugin...")

		err = downloadBinary(ctx, binPath, url, true)
		if err != nil {
			return fmt.Errorf("while downloading dependency from URL %q (checksum: %q): %w", url.URL, url.Checksum, err)
		}
	}

	// Ensure all dependencies are downloaded
	log.Info("Ensuring plugin dependencies are downloaded...")
	depDir := dependencyDirForBin(binPath)
	for depName, dep := range info.Dependencies {
		depPath := filepath.Join(depDir, depName)
		if DoesBinaryExist(depPath) {
			m.log.Debugf("Binary %q found locally. Skipping...", depName)
			continue
		}

		depURL, found := dep[selector]
		if !found {
			return NewNotFoundPluginError("cannot find download url for current platform for a dependency %q of the plugin %q", depName, binPath)
		}

		log.WithFields(logrus.Fields{
			"dependencyName": depName,
			"dependencyUrl":  depURL,
		}).Info("Downloading dependency...")

		err := downloadBinary(ctx, depPath, URL{URL: depURL}, false)
		if err != nil {
			return fmt.Errorf("while downloading dependency %q for %q: %w", depName, binPath, err)
		}
	}

	return nil
}

func dependencyDirForBin(binPath string) string {
	return fmt.Sprintf("%s_deps", binPath)
}

// DoesBinaryExist returns true if a given file exists.
func DoesBinaryExist(path string) bool {
	stat, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}

	return err == nil && !stat.IsDir()
}
