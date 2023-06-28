package x

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

var selectIndicatorFinder = regexp.MustCompile(fmt.Sprintf(`%s(\d+)`, SelectIndexIndicator))
var pageIndicatorFinder = regexp.MustCompile(fmt.Sprintf(`%s(\d+)`, PageIndexIndicator))

const (
	RawOutputIndicator   = "@raw"
	SelectIndexIndicator = "@idx:"
	PageIndexIndicator   = "@page:"
)

type Command struct {
	ToExecute     string
	IsRawRequired bool
	PageIndex     int
}

func Parse(cmd string) Command {
	out := Command{
		ToExecute: cmd,
	}
	if strings.Contains(out.ToExecute, RawOutputIndicator) {
		out.ToExecute = strings.ReplaceAll(out.ToExecute, RawOutputIndicator, "")
		out.IsRawRequired = true
	}
	groups := pageIndicatorFinder.FindAllStringSubmatch(cmd, -1)
	if len(groups) > 0 && len(groups[0]) > 1 {
		out.PageIndex, _ = strconv.Atoi(groups[0][1])
	}

	out.ToExecute = selectIndicatorFinder.ReplaceAllString(out.ToExecute, "")
	out.ToExecute = pageIndicatorFinder.ReplaceAllString(out.ToExecute, "")
	out.ToExecute = strings.TrimSpace(out.ToExecute)

	return out
}
