package utils

import (
	"fmt"
	"strings"

	"k8s.io/client-go/util/jsonpath"
	"k8s.io/kubectl/pkg/cmd/get"
)

func parseJsonpath(obj interface{}, jsonpathStr string) (string, error) {
	// Parse and print jsonpath
	fields, err := get.RelaxedJSONPathExpression(jsonpathStr)
	if err != nil {
		return "", err
	}

	j := jsonpath.New("jsonpath")
	if err := j.Parse(fields); err != nil {
		return "", err
	}

	values, err := j.FindResults(obj)

	valueStrings := []string{}
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
