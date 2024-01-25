package pluginx

import (
	"github.com/kubeshop/botkube/internal/plugin"
	"github.com/sirupsen/logrus"
)

// NewIndexBuilder returns a new IndexBuilder instance.
func NewIndexBuilder(log logrus.FieldLogger) *plugin.IndexBuilder {
	return plugin.NewIndexBuilder(log)
}
