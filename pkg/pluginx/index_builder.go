package pluginx

import (
	"github.com/sirupsen/logrus"

	"github.com/kubeshop/botkube/internal/plugin"
)

// NewIndexBuilder returns a new IndexBuilder instance.
func NewIndexBuilder(log logrus.FieldLogger) *plugin.IndexBuilder {
	return plugin.NewIndexBuilder(log)
}
