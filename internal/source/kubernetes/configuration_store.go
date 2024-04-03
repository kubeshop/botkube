package kubernetes

import (
	"fmt"
	"sync"

	"github.com/kubeshop/botkube/pkg/maputil"
)

// configurationStore stores all source configurations in a thread-safe way.
type configurationStore struct {
	store             map[string]SourceConfig
	storeByKubeconfig map[string]map[string]SourceConfig

	lock sync.RWMutex
}

// newConfigurations creates new empty configurationStore instance.
func newConfigurations() *configurationStore {
	return &configurationStore{
		store:             make(map[string]SourceConfig),
		storeByKubeconfig: make(map[string]map[string]SourceConfig),
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
		c.storeByKubeconfig[kubeConfigKey] = make(map[string]SourceConfig)
	}
	c.storeByKubeconfig[kubeConfigKey][key] = cfg
}

// Get returns SourceConfig by a key.
func (c *configurationStore) Get(sourceKey string) (SourceConfig, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	val, ok := c.store[sourceKey]
	return val, ok
}

// GetGlobal returns global SourceConfig.
func (c *configurationStore) GetGlobal() (SourceConfig, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	sortedKeys := c.sortedKeys()
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

	cloned := make(map[string]map[string]SourceConfig)
	for k, v := range c.storeByKubeconfig {
		cloned[k] = v
	}

	return cloned
}

func (c *configurationStore) sortedKeys() []string {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return maputil.SortKeys(c.store)
}

func (c *configurationStore) keyForStore(sourceName string, isInteractivitySupported bool) string {
	return fmt.Sprintf("%s/%t", sourceName, isInteractivitySupported)
}
