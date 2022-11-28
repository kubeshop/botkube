package plugin

import (
	"fmt"
	"sort"

	semver "github.com/hashicorp/go-version"
	"gopkg.in/yaml.v3"
)

type (
	// store holds information about processed and started plugins.
	store[T any] struct {
		Repository     storeRepository
		EnabledPlugins storePlugins[T]
	}

	// storeRepository holds known plugins metadata indexed by {repo}/{plugin_name} key.
	// Additionally, all entries for a given key are sorted by Version field.
	storeRepository map[string][]storeEntry

	storeEntry struct {
		Description string
		Version     string
		URLs        map[string]string
	}

	// storePlugins holds enabled plugins indexed by {repo}/{plugin_name} key.
	storePlugins[T any] map[string]enabledPlugins[T]

	enabledPlugins[T any] struct {
		Client  T
		Cleanup func()
	}
)

func newStore[T any]() store[T] {
	return store[T]{
		Repository:     storeRepository{},
		EnabledPlugins: map[string]enabledPlugins[T]{},
	}
}

func newStoreRepositories(indexes map[string][]byte) (storeRepository, storeRepository, error) {
	var (
		executorsRepositories = storeRepository{}
		sourcesRepositories   = storeRepository{}
	)

	for repo, data := range indexes {
		var index Index
		if err := yaml.Unmarshal(data, &index); err != nil {
			fmt.Println(string(data))
			return nil, nil, fmt.Errorf("while unmarshaling index: %w", err)
		}

		for _, entry := range index.Entries {
			// omit version, as we want to collect plugins with different version together
			key, err := BuildPluginKey(repo, entry.Name, "")
			if err != nil {
				return nil, nil, fmt.Errorf("while building key for entry in %s repository: %w", repo, err)
			}

			switch entry.Type {
			case TypeExecutor:
				executorsRepositories[key] = append(executorsRepositories[key], storeEntry{
					Description: entry.Description,
					Version:     entry.Version,
					URLs:        mapBinaryURLs(entry.URLs),
				})
			case TypeSource:
				sourcesRepositories[key] = append(sourcesRepositories[key], storeEntry{
					Description: entry.Description,
					Version:     entry.Version,
					URLs:        mapBinaryURLs(entry.URLs),
				})
			}
		}
	}

	// sort loaded entries by version
	for key := range executorsRepositories {
		sort.Sort(byIndexEntryVersion(executorsRepositories[key]))
	}

	for key := range sourcesRepositories {
		sort.Sort(byIndexEntryVersion(sourcesRepositories[key]))
	}

	return executorsRepositories, sourcesRepositories, nil
}

func mapBinaryURLs(in []IndexURL) map[string]string {
	out := map[string]string{}
	for _, item := range in {
		key := item.Platform.OS + "/" + item.Platform.Arch
		out[key] = item.URL
	}
	return out
}

// byIndexEntryVersion implements sort.Interface based on the version field.
type byIndexEntryVersion []storeEntry

func (a byIndexEntryVersion) Len() int      { return len(a) }
func (a byIndexEntryVersion) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a byIndexEntryVersion) Less(i, j int) bool {
	return semvVerAGreaterThanB(a[i].Version, a[j].Version)
}

// semvVerAGreaterThanB returns true if A is greater than B.
func semvVerAGreaterThanB(a, b string) bool {
	verA, aErr := semver.NewVersion(a)
	verB, bErr := semver.NewVersion(b)

	return aErr == nil && bErr == nil && verA.GreaterThan(verB)
}
