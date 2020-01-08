package utils

import (
	"fmt"

	"github.com/infracloudio/botkube/pkg/config"
)

type diffReporter struct {
	field string
}

func (d diffReporter) exec(x, y interface{}) (string, bool) {
	vx, err := parseJsonpath(x, d.field)
	if err != nil {
		return "", false
	}

	vy, err := parseJsonpath(y, d.field)
	if err != nil {
		return "", false
	}

	// treat <none> and false as same fields
	if vx == vy || (vx == "<none>" && vy == "false") {
		return "", false
	}
	return fmt.Sprintf("%s:\n\t-: %+v\n\t+: %+v\n", d.field, vx, vy), true
}

// Diff provides differences between two objects spec
func Diff(x, y interface{}, updatesetting config.UpdateSetting) string {
	msg := ""
	for _, val := range updatesetting.Fields {
		var d diffReporter
		d.field = val
		if diff, ok := d.exec(x, y); ok {
			msg = msg + diff
		}
	}
	return msg
}
