package kubernetes

import (
	"fmt"
	"github.com/kubeshop/botkube/pkg/maputil"
	"sync"
)

type ConfigurationStore struct {
	store             map[string]SourceConfig
	storeByKubeconfig map[string]map[string]SourceConfig

	lock sync.RWMutex
}

func NewConfigurations() *ConfigurationStore {
	return &ConfigurationStore{
		store:             make(map[string]SourceConfig),
		storeByKubeconfig: make(map[string]map[string]SourceConfig),
	}
}

func (c *ConfigurationStore) Store(sourceName string, cfg SourceConfig) {
	c.lock.Lock()
	defer c.lock.Unlock()

	key := keyForStore(sourceName, cfg.isInteractivitySupported)

	c.store[key] = cfg

	kubeConfigKey := string(cfg.kubeConfig)
	if _, ok := c.storeByKubeconfig[kubeConfigKey]; !ok {
		c.storeByKubeconfig[kubeConfigKey] = make(map[string]SourceConfig)
	}
	c.storeByKubeconfig[kubeConfigKey][key] = cfg
}

func (c *ConfigurationStore) Get(sourceKey string) (SourceConfig, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	val, ok := c.store[sourceKey]
	return val, ok
}

func (c *ConfigurationStore) GetGlobal() (SourceConfig, bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()

	sortedKeys := c.sortedKeys()
	if len(sortedKeys) == 0 {
		return SourceConfig{}, false
	}

	return c.store[sortedKeys[0]], true
}

func (c *ConfigurationStore) Len() int {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return len(c.store)
}

func (c *ConfigurationStore) sortedKeys() []string {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return maputil.SortKeys(c.store)
}

func (c *ConfigurationStore) CloneByKubeconfig() map[string]map[string]SourceConfig {
	c.lock.RLock()
	defer c.lock.RUnlock()

	cloned := make(map[string]map[string]SourceConfig)
	for k, v := range c.storeByKubeconfig {
		cloned[k] = v
	}

	return cloned
}

func keyForStore(sourceName string, isInteractivitySupported bool) string {
	return fmt.Sprintf("%s/%t", sourceName, isInteractivitySupported)
}
