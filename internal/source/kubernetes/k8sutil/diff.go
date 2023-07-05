package k8sutil

import (
	"fmt"
	"strings"

	"k8s.io/client-go/util/jsonpath"
	"k8s.io/kubectl/pkg/cmd/get"

	"github.com/kubeshop/botkube/internal/source/kubernetes/config"
	"github.com/kubeshop/botkube/pkg/multierror"
)

// Diff provides differences between two objects.
func Diff(x, y interface{}, updateSetting config.UpdateSetting) (string, error) {
	strBldr := new(strings.Builder)

	errs := multierror.New()
	for _, val := range updateSetting.Fields {
		var d diffReporter
		d.field = val
		diff, err := d.exec(x, y)
		if err != nil {
			errs = multierror.Append(errs, err)
			continue
		}

		strBldr.WriteString(diff)
	}

	if errs.ErrorOrNil() != nil {
		return strBldr.String(), fmt.Errorf("while getting diff: %w", errs.ErrorOrNil())
	}

	return strBldr.String(), nil
}

type diffReporter struct {
	field string
}

func (d diffReporter) exec(x, y interface{}) (string, error) {
	vx, err := parseJsonpath(x, d.field)
	if err != nil {
		return "", fmt.Errorf("while finding value in old obj from jsonpath %q: %w", d.field, err)
	}

	vy, err := parseJsonpath(y, d.field)
	if err != nil {
		return "", fmt.Errorf("while finding value in new obj from jsonpath %q: %w", d.field, err)
	}

	// treat <none> and false as same fields
	if vx == vy || (vx == "<none>" && vy == "false") {
		return "", nil
	}
	return fmt.Sprintf("%s:\n\t-: %+v\n\t+: %+v\n", d.field, vx, vy), nil
}

func parseJsonpath(obj interface{}, jsonpathStr string) (string, error) {
	// Parse and print jsonpath
	fields, err := get.RelaxedJSONPathExpression(jsonpathStr)
	if err != nil {
		return "", err
	}

	j := jsonpath.New("jsonpath")
	j.AllowMissingKeys(true)
	if err := j.Parse(fields); err != nil {
		return "", err
	}

	values, err := j.FindResults(obj)
	if err != nil {
		return "", err
	}

	var valueStrings []string
	if len(values) == 0 || len(values[0]) == 0 {
		valueStrings = append(valueStrings, "<none>")
	}
	for arrIx := range values {
		for valIx := range values[arrIx] {
			valueStrings = append(valueStrings, fmt.Sprintf("%v", values[arrIx][valIx].Interface()))
		}
	}
	return strings.Join(valueStrings, ","), nil
}
