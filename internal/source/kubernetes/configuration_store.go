package kubernetes

import (
	"fmt"
	"sync"

	"github.com/kubeshop/botkube/pkg/maputil"
)

// configurationStore stores all source configurations in a thread-safe way.
type configurationStore struct {
	store             map[string]SourceConfig
	storeByKubeconfig map[string]map[string]struct{}

	lock sync.RWMutex
}

// newConfigurations creates new empty configurationStore instance.
func newConfigurations() *configurationStore {
	return &configurationStore{
		store:             make(map[string]SourceConfig),
		storeByKubeconfig: make(map[string]map[string]struct{}),
	}
}

// Store stores SourceConfig in a thread-safe way.
func (c *configurationStore) Store(sourceName string, cfg SourceConfig) {
	c.lock.Lock()
	defer c.lock.Unlock()

	key := c.keyForStore(sourceName, cfg.isInteractivitySupported)

	c.store[key] = cfg

	kubeConfigKey := string(cfg.kubeConfig)
	if _, ok := c.storeByKubeconfig[kubeConfigKey]; !ok {
		c.storeByKubeconfig[kubeConfigKey] = make(map[string]struct{})
	}
	c.storeByKubeconfig[kubeConfigKey][key] = struct{}{}
}

// Get returns SourceConfig by a key.
func (c *configurationStore) Get(sourceKey string) (SourceConfig, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	val, ok := c.store[sourceKey]
	return val, ok
}

// GetSystemConfig returns system Source Config.
// The system config is used for getting system (plugin-wide) logger and informer resync period.
func (c *configurationStore) GetSystemConfig() (SourceConfig, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	sortedKeys := maputil.SortKeys(c.store)
	if len(sortedKeys) == 0 {
		return SourceConfig{}, false
	}

	return c.store[sortedKeys[0]], true
}

// Len returns number of stored SourceConfigs.
func (c *configurationStore) Len() int {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return len(c.store)
}

// CloneByKubeconfig returns a copy of the underlying map of source configurations grouped by kubeconfigs.
func (c *configurationStore) CloneByKubeconfig() map[string]map[string]SourceConfig {
	c.lock.RLock()
	defer c.lock.RUnlock()

	var out = make(map[string]map[string]SourceConfig)
	for kubeConfig, srcIndex := range c.storeByKubeconfig {
		if out[kubeConfig] == nil {
			out[kubeConfig] = make(map[string]SourceConfig)
		}

		for srcKey := range srcIndex {
			out[kubeConfig][srcKey] = c.store[srcKey]
		}
	}

	return out
}

// keyForStore returns a key for storing configuration in the store.
func (c *configurationStore) keyForStore(sourceName string, isInteractivitySupported bool) string {
	return fmt.Sprintf("%s/%t", sourceName, isInteractivitySupported)
}
