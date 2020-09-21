// Copyright (c) 2020 InfraCloud Technologies
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies of
// the Software, and to permit persons to whom the Software is furnished to do so,
// subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY, FITNESS
// FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR
// COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER
// IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN
// CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package utils

import (
	"fmt"

	"github.com/infracloudio/botkube/pkg/config"
	"github.com/infracloudio/botkube/pkg/log"
)

type diffReporter struct {
	field string
}

func (d diffReporter) exec(x, y interface{}) (string, bool) {
	vx, err := parseJsonpath(x, d.field)
	if err != nil {
		log.Debugf("Failed to find value from jsonpath: %s, object: %+v. Error: %v", d.field, x, err)
		return "", false
	}

	vy, err := parseJsonpath(y, d.field)
	if err != nil {
		log.Debugf("Failed to find value from jsonpath: %s, object: %+v, Error: %v", d.field, y, err)
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
