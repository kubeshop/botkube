package plugin

// Type represents the plugin type.
type Type string

const (
	// TypeSource represents the source plugin.
	TypeSource Type = "source"
	// TypeExecutor represents the executor plugin.
	TypeExecutor Type = "executor"
)

type (
	// Index defines the plugin repository index.
	Index struct {
		Entries []IndexEntry `yaml:"entries"`
	}
	// IndexEntry defines the plugin definition.
	IndexEntry struct {
		Name        string     `yaml:"name"`
		Type        Type       `yaml:"type"`
		Description string     `yaml:"description"`
		Version     string     `yaml:"version"`
		URLs        []IndexURL `yaml:"urls"`
	}

	// IndexURL holds the binary url details.
	IndexURL struct {
		URL      string           `yaml:"url"`
		Platform IndexURLPlatform `yaml:"platform"`
	}

	// IndexURLPlatform holds platform information about a given binary URL.
	IndexURLPlatform struct {
		OS   string `yaml:"os"`
		Arch string `yaml:"architecture"`
	}
)
