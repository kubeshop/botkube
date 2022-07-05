package utils

import (
	"fmt"
	"strings"

	"github.com/kubeshop/botkube/pkg/config"
)

type diffReporter struct {
	field string
}

func (d diffReporter) exec(x, y interface{}) (string, error) {
	vx, err := parseJsonpath(x, d.field)
	if err != nil {
		// Happens when the fields were not set by the time event was issued, do not return in that case
		return "", fmt.Errorf("while finding value from jsonpath: %q, object: %+v: %w", d.field, x, err)
	}

	vy, err := parseJsonpath(y, d.field)
	if err != nil {
		return "", fmt.Errorf("while finding value from jsonpath: %q, object: %+v: %w", d.field, y, err)
	}

	// treat <none> and false as same fields
	if vx == vy || (vx == "<none>" && vy == "false") {
		return "", nil
	}
	return fmt.Sprintf("%s:\n\t-: %+v\n\t+: %+v\n", d.field, vx, vy), nil
}

// Diff provides differences between two objects spec
func Diff(x, y interface{}, updateSetting config.UpdateSetting) (string, error) {
	strBldr := new(strings.Builder)
	for _, val := range updateSetting.Fields {
		var d diffReporter
		d.field = val
		diff, err := d.exec(x, y)
		if err != nil {
			return "", err
		}

		strBldr.WriteString(diff)
	}

	return strBldr.String(), nil
}
