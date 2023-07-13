package logs

import (
	"encoding/json"
	"fmt"
	"sort"

	"golang.org/x/exp/maps"

	charmlog "github.com/charmbracelet/log"

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

	lvl := charmlog.ParseLevel(fmt.Sprint(result["level"]))
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
