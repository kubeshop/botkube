package logs

import (
	"encoding/json"
	"fmt"
	"sort"

	"golang.org/x/exp/maps"

	charmlog "github.com/charmbracelet/log"

	"github.com/kubeshop/botkube/internal/cli"
)

type JSONParser struct{}

func (p *JSONParser) ParseLine(line string) map[string]any {
	var out map[string]any
	err := json.Unmarshal([]byte(line), &out)
	if err != nil {
		return nil
	}
	return out
}

// ParseLineIntoCharm returns parsed log line with charm logger support.
func (k *JSONParser) ParseLineIntoCharm(line string) ([]any, charmlog.Level) {
	result := k.ParseLine(line)
	if result == nil {
		return nil, 0
	}

	var fields []any

	//if k.ReportTimestamp {
	//	parseAny, _ := dateparse.ParseAny(result["time"])
	//	fields = append(fields, charmlog.TimestampKey, parseAny)
	//}

	lvl := charmlog.ParseLevel(fmt.Sprint(result["level"]))
	// todo, check and ignore debug
	fields = append(fields, charmlog.LevelKey, lvl)
	fields = append(fields, charmlog.MessageKey, result["msg"])

	keys := maps.Keys(result)
	sort.Strings(keys)
	for _, k := range keys {
		switch k {
		case "level", "msg", "time": // already process
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
