package loggerx

import (
	"github.com/hashicorp/go-hclog"
	"github.com/sirupsen/logrus"
	"github.com/spiffe/spire/pkg/common/log"
)

// AsHCLog return logger that implements the hclog interface, and wraps it around a Logrus entry.
func AsHCLog(logger logrus.FieldLogger, name string) hclog.Logger {
	return log.NewHCLogAdapter(logger, name)
}
