package logs

import (
	"regexp"
	"sort"
	"strings"

	"github.com/araddon/dateparse"
	charmlog "github.com/charmbracelet/log"
	"golang.org/x/exp/maps"
)

// Regex pattern to match key-value pairs
var kvMatcher = regexp.MustCompile(`(\w+)(?:="([^"]+)"|=([^"\s]+))?`)

// KVParser knows how to parse key-value log pairs.
type KVParser struct {
	ReportTimestamp bool
}

func NewKVParser(reportTimestamp bool) *KVParser {
	return &KVParser{ReportTimestamp: reportTimestamp}
}

// ParseLineIntoCharm returns parsed log line with charm logger support.
func (k *KVParser) ParseLineIntoCharm(line string) []any {
	result := k.ParseLine(line)

	var fields []any

	if k.ReportTimestamp {
		parseAny, _ := dateparse.ParseAny(result["time"])
		fields = append(fields, charmlog.TimestampKey, parseAny)
	}

	fields = append(fields, charmlog.LevelKey, charmlog.ParseLevel(result["level"]))
	fields = append(fields, charmlog.MessageKey, result["msg"])

	keys := maps.Keys(result)
	sort.Strings(keys)
	for _, k := range keys {
		switch k {
		case "level", "msg", "time": // already process
			continue
		}
		fields = append(fields, k, result[k])
	}

	return fields
}

func (k *KVParser) ParseLine(line string) map[string]string {
	result := make(map[string]string)

	matches := kvMatcher.FindAllStringSubmatch(line, -1)
	for _, match := range matches {
		m := safeSliceGetter(match)
		key := m.Get(1)
		value := strings.TrimSpace(m.Get(2))
		if value == "" {
			value = strings.TrimSpace(m.Get(3))
		}
		result[key] = value
	}

	return result
}

type safeSliceGetter []string

func (s safeSliceGetter) Get(idx int) string {
	if len(s) < idx {
		return ""
	}
	return s[idx]
}
