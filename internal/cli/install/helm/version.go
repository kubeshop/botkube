package helm

import (
	"fmt"
	"github.com/kubeshop/botkube/pkg/httpx"
	"io"
	"net/url"

	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/repo"
	"sigs.k8s.io/yaml"
)

// GetLatestVersion loads an index file and returns version of the latest chart. Sort by SemVer.
//
// Assumption that all charts are versioned in the same way.
func GetLatestVersion(repoURL string, chart string) (string, error) {
	path, err := url.JoinPath(repoURL, "index.yaml")
	if err != nil {
		return "", err
	}

	httpClient := httpx.NewHTTPClient()
	resp, err := httpClient.Get(path)
	if err != nil {
		return "", fmt.Errorf("while getting Botkube Helm Chart repository index.yaml: %v", err)
	}
	defer resp.Body.Close()

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("while reading response body: %v", err)
	}

	i := &repo.IndexFile{}
	if err := yaml.UnmarshalStrict(data, i); err != nil {
		return "", errors.Wrapf(err, "Index fetch from %q is malformed", path)
	}

	// by default sort by SemVer, so even if someone pushed bugfix of older
	// release we will not take it.
	i.SortEntries()

	entry, ok := i.Entries[chart]
	if !ok {
		return "", fmt.Errorf("no entry %q in Helm Chart repository index.yaml", chart)
	}

	if len(entry) == 0 {
		return "", fmt.Errorf("no Chart versions for entry %q in Helm Chart repository index.yaml", chart)
	}

	return entry[0].Version, nil
}
