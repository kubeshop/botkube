package plugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/pkg/api"
)

type metadataGetter interface {
	Metadata(context.Context) (api.MetadataOutput, error)
}

type pluginBinariesIndex struct {
	BinaryPath string
	OS         string
	Arch       string
	Type       Type
}

// IndexBuilder provides functionality to generate plugin index.
type IndexBuilder struct {
	log logrus.FieldLogger
}

// NewIndexBuilder returns a new IndexBuilder instance.
func NewIndexBuilder(log logrus.FieldLogger) *IndexBuilder {
	return &IndexBuilder{
		log: log.WithField("service", "Plugin Index Builder"),
	}
}

// Build returns plugin index built based on plugins found in a given directory.
func (i *IndexBuilder) Build(dir, urlBasePath string) (Index, error) {
	files, err := os.ReadDir(dir)
	if err != nil {
		return Index{}, fmt.Errorf("while reading directory with plugin binaries: %w", err)
	}

	// group by {plugin_type}_{plugin_name}
	entries := map[string][]pluginBinariesIndex{}
	for _, entry := range files {
		if entry.IsDir() {
			continue
		}

		err := i.appendIndexEntry(entries, entry.Name())
		if err != nil {
			return Index{}, fmt.Errorf("while adding executor entry: %w", err)
		}
	}

	var out Index
	for key, bins := range entries {
		meta, err := i.getPluginMetadata(dir, bins)
		if err != nil {
			return Index{}, fmt.Errorf("while getting plugin metadata: %w", err)
		}

		pType, pName, _ := strings.Cut(key, "/")
		out.Entries = append(out.Entries, IndexEntry{
			Name:        pName,
			Type:        Type(pType),
			Description: meta.Description,
			Version:     meta.Version,
			JSONSchema:  meta.JSONSchema,
			URLs:        i.mapToIndexURLs(bins, urlBasePath),
		})
	}
	return out, nil
}

func (*IndexBuilder) mapToIndexURLs(bins []pluginBinariesIndex, urlBasePath string) []IndexURL {
	var urls []IndexURL
	for _, bin := range bins {
		urls = append(urls, IndexURL{
			URL: fmt.Sprintf("%s/%s", urlBasePath, bin.BinaryPath),
			Platform: IndexURLPlatform{
				OS:   bin.OS,
				Arch: bin.Arch,
			},
		})
	}

	return urls
}

func (i *IndexBuilder) getPluginMetadata(dir string, bins []pluginBinariesIndex) (*api.MetadataOutput, error) {
	os, arch := runtime.GOOS, runtime.GOARCH

	for _, item := range bins {
		if item.Arch != arch || item.OS != os {
			continue
		}

		bins := map[string]string{
			item.Type.String(): filepath.Join(dir, item.BinaryPath),
		}
		clients, err := createGRPCClients[metadataGetter](i.log, bins, item.Type)
		if err != nil {
			return nil, fmt.Errorf("while creating gRPC client: %w", err)
		}

		cli := clients[item.Type.String()]
		meta, err := cli.Client.Metadata(context.Background())
		if err != nil {
			return nil, fmt.Errorf("while calling metadata RPC: %w", err)
		}
		cli.Cleanup()

		fmt.Printf("Schema: '%s'\n", meta.JSONSchema)

		if err := meta.Validate(); err != nil {
			return nil, fmt.Errorf("while validating metadata fields: %w", err)
		}

		return &meta, nil
	}

	return nil, fmt.Errorf("cannot find binary for %s/%s", os, arch)
}

func (i *IndexBuilder) appendIndexEntry(entries map[string][]pluginBinariesIndex, entryName string) error {
	if !strings.HasPrefix(entryName, TypeExecutor.String()) && !strings.HasPrefix(entryName, TypeSource.String()) {
		i.log.WithField("file", entryName).Debug("Ignoring file as not recognized as plugin")
		return nil
	}

	parts := strings.Split(entryName, "_")
	if len(parts) != 4 {
		return fmt.Errorf("path %s doesn't follow required pattern <plugin_type>_<plugin_name>_<os>_<arch>", entryName)
	}

	pType, pName, os, arch := parts[0], parts[1], parts[2], parts[3]
	i.log.WithFields(logrus.Fields{
		"type": pType,
		"name": pName,
		"os":   os,
		"arch": arch,
	}).Debug("Indexing plugin...")

	key := fmt.Sprintf("%s/%s", pType, pName)
	entries[key] = append(entries[key], pluginBinariesIndex{
		BinaryPath: entryName,
		OS:         os,
		Type:       Type(pType),
		Arch:       arch,
	})

	return nil
}
