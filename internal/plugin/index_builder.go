package plugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/xeipuuv/gojsonschema"

	"github.com/kubeshop/botkube/pkg/api"
	"github.com/kubeshop/botkube/pkg/multierror"
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
func (i *IndexBuilder) Build(dir, urlBasePath, pluginNameFilter string) (Index, error) {
	pluginNameRegex, err := regexp.Compile(pluginNameFilter)
	if err != nil {
		return Index{}, fmt.Errorf("while compiling filter regex: %w", err)
	}

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

		err := i.appendIndexEntry(entries, entry.Name(), pluginNameRegex)
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
			JSONSchema: JSONSchema{
				Value:  meta.JSONSchema.Value,
				RefURL: meta.JSONSchema.RefURL,
			},
			URLs: i.mapToIndexURLs(bins, urlBasePath, meta.Dependencies),
		})
	}

	i.log.Info("Validating JSON schemas...")
	err = i.validateJSONSchemas(out)
	if err != nil {
		return Index{}, fmt.Errorf("while validating JSON schemas: %w", err)
	}

	return out, nil
}

func (i *IndexBuilder) mapToIndexURLs(bins []pluginBinariesIndex, urlBasePath string, deps map[string]api.Dependency) []IndexURL {
	var urls []IndexURL
	for _, bin := range bins {
		urls = append(urls, IndexURL{
			URL: fmt.Sprintf("%s/%s", urlBasePath, bin.BinaryPath),
			Platform: IndexURLPlatform{
				OS:   bin.OS,
				Arch: bin.Arch,
			},
			Dependencies: i.dependenciesForBinary(bin, deps),
		})
	}

	return urls
}

func (*IndexBuilder) dependenciesForBinary(bin pluginBinariesIndex, deps map[string]api.Dependency) Dependencies {
	out := make(Dependencies)
	for depName, depDetails := range deps {
		url, exists := depDetails.URLs.For(bin.OS, bin.Arch)
		if !exists {
			continue
		}

		out[depName] = Dependency{
			URL: url,
		}
	}

	return out
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

		if err := meta.Validate(); err != nil {
			return nil, fmt.Errorf("while validating metadata fields: %w", err)
		}

		return &meta, nil
	}

	return nil, fmt.Errorf("cannot find binary for %s/%s", os, arch)
}

func (i *IndexBuilder) appendIndexEntry(entries map[string][]pluginBinariesIndex, entryName string, pNameRegex *regexp.Regexp) error {
	if !strings.HasPrefix(entryName, TypeExecutor.String()) && !strings.HasPrefix(entryName, TypeSource.String()) {
		i.log.WithField("file", entryName).Debug("Ignoring file as not recognized as plugin")
		return nil
	}

	parts := strings.Split(entryName, "_")
	if len(parts) != 4 {
		return fmt.Errorf("path %s doesn't follow required pattern <plugin_type>_<plugin_name>_<os>_<arch>", entryName)
	}

	pType, pName, os, arch := parts[0], parts[1], parts[2], parts[3]
	if !pNameRegex.MatchString(pName) {
		i.log.WithField("file", entryName).Debug("Ignoring file as it doesn't match filter")
		return nil
	}

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

const jsonSchemaSpecURL = "https://json-schema.org/draft-07/schema"

func (i *IndexBuilder) validateJSONSchemas(in Index) error {
	schemaLoader := gojsonschema.NewReferenceLoader(jsonSchemaSpecURL)
	schemaDraft07, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		return fmt.Errorf("while loading JSON schema draft-07: %w", err)
	}

	errs := multierror.New()
	for _, entry := range in.Entries {
		entrySchemaLoader := i.getJSONSchemaLoaderForEntry(entry)
		if err != nil {
			wrappedErr := fmt.Errorf("while loading JSON schema for %s: %w", entry.Name, err)
			errs = multierror.Append(errs, wrappedErr)
			continue
		}
		if entrySchemaLoader == nil {
			continue
		}

		jsonSchemaValidationResult, err := schemaDraft07.Validate(entrySchemaLoader)
		if err != nil {
			wrappedErr := fmt.Errorf("while validating JSON schema for %s: %w", entry.Name, err)
			errs = multierror.Append(errs, wrappedErr)
			continue
		}

		for _, err := range jsonSchemaValidationResult.Errors() {
			wrappedErr := fmt.Errorf("while validating JSON schema for %s: %s", entry.Name, err)
			errs = multierror.Append(errs, wrappedErr)
		}
	}

	return errs.ErrorOrNil()
}

func (i *IndexBuilder) getJSONSchemaLoaderForEntry(entry IndexEntry) gojsonschema.JSONLoader {
	if entry.JSONSchema.Value != "" {
		return gojsonschema.NewStringLoader(entry.JSONSchema.Value)
	}

	refURL := entry.JSONSchema.RefURL
	if refURL != "" {
		return gojsonschema.NewReferenceLoader(refURL)
	}

	return nil
}
