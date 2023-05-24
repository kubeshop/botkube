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
		Description  string
		Version      string
		URLs         map[string]URL
		Dependencies map[string]map[string]string
		JSONSchema   JSONSchema
	}

	URL struct {
		URL      string
		Checksum string
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
			return nil, nil, fmt.Errorf("while unmarshaling index: %w", err)
		}

		if err := index.Validate(); err != nil {
			return nil, nil, fmt.Errorf("while validating %s index: %w", repo, err)
		}

		for _, entry := range index.Entries {
			binURLs, depURLs := mapBinaryURLs(entry.URLs)

			switch entry.Type {
			case TypeExecutor:
				executorsRepositories.Insert(repo, entry.Name, storeEntry{
					Description:  entry.Description,
					Version:      entry.Version,
					URLs:         binURLs,
					Dependencies: depURLs,
					JSONSchema:   entry.JSONSchema,
				})
			case TypeSource:
				sourcesRepositories.Insert(repo, entry.Name, storeEntry{
					Description:  entry.Description,
					Version:      entry.Version,
					URLs:         binURLs,
					Dependencies: depURLs,
					JSONSchema:   entry.JSONSchema,
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

func (s storeRepository) Insert(repo, name string, entry storeEntry) {
	if s == nil {
		return
	}
	key := s.key(repo, name)

	s[key] = append(s[key], entry)
}

func (s storeRepository) Get(repo, name string) ([]storeEntry, bool) {
	if s == nil {
		return nil, false
	}

	key := s.key(repo, name)
	entry, found := s[key]
	return entry, found
}

// omit version, as we want to collect plugins with different version together
func (s storeRepository) key(repo, name string) string {
	return fmt.Sprintf("%s/%s", repo, name)
}

func mapBinaryURLs(in []IndexURL) (map[string]URL, map[string]map[string]string) {
	out := make(map[string]URL)
	var deps map[string]map[string]string
	for _, item := range in {
		key := item.Platform.OS + "/" + item.Platform.Arch
		out[key] = URL{
			URL:      item.URL,
			Checksum: item.Checksum,
		}

		for depName, dep := range item.Dependencies {
			if deps == nil {
				deps = make(map[string]map[string]string)
			}

			if deps[depName] == nil {
				deps[depName] = make(map[string]string)
			}

			deps[depName][key] = dep.URL
		}
	}

	return out, deps
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
