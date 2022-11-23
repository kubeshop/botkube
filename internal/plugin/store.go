package plugin

type (
	// store holds information about processed and started plugins.
	store[T any] struct {
		Repository     storeRepository
		EnabledPlugins storePlugins[T]
	}

	// storeRepository holds known plugins metadata indexed by {repo}/{plugin_name} key.
	// Additionally, all entries for a given key are sorted by Version field.
	storeRepository map[string][]IndexEntry

	// storePlugins holds enabled plugins indexed by {repo}/{plugin_name} key.
	storePlugins[T any] map[string]enabledPlugins[T]

	enabledPlugins[T any] struct {
		Client  T
		Cleanup func()
	}
)

func newStore[T any]() store[T] {
	return store[T]{
		Repository:     map[string][]IndexEntry{},
		EnabledPlugins: map[string]enabledPlugins[T]{},
	}
}
