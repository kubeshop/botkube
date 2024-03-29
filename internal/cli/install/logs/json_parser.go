package logs

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	charmlog "github.com/charmbracelet/log"
	"golang.org/x/exp/maps"

	"github.com/kubeshop/botkube/internal/cli"
)

// JSONParser knows how to parse JSON formatted logs.
type JSONParser struct{}

// ParseLineIntoCharm returns parsed log line with charm logger support.
func (j *JSONParser) ParseLineIntoCharm(line string) ([]any, charmlog.Level) {
	result := j.parseLine(line)
	if result == nil {
		return nil, 0
	}

	var fields []any

	lvl := parseLevel(fmt.Sprint(result["level"]))
	fields = append(fields, charmlog.LevelKey, lvl)
	fields = append(fields, charmlog.MessageKey, result["msg"])

	keys := maps.Keys(result)
	sort.Strings(keys)
	for _, k := range keys {
		switch k {
		case "level", "msg", "time": // already processed
			continue
		case "component", "url":
			if !cli.VerboseMode.IsEnabled() {
				continue // ignore those fields if verbose is not enabled
			}
		}
		fields = append(fields, k, result[k])
	}

	return fields, lvl
}

func (*JSONParser) parseLine(line string) map[string]any {
	var out map[string]any
	err := json.Unmarshal([]byte(line), &out)
	if err != nil {
		return nil
	}
	return out
}

// parseLevel takes a string level and returns the charm log level constant.
func parseLevel(lvl string) charmlog.Level {
	switch strings.ToLower(lvl) {
	case "panic", "fatal":
		return charmlog.FatalLevel
	case "error", "err":
		return charmlog.ErrorLevel
	case "warn", "warning":
		return charmlog.WarnLevel
	case "info":
		return charmlog.InfoLevel
	case "debug", "trace":
		return charmlog.DebugLevel
	default:
		return charmlog.InfoLevel
	}
}
