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
	"strings"

	"github.com/infracloudio/botkube/pkg/config"
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
