package github_events

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
	"k8s.io/client-go/util/jsonpath"
	"k8s.io/kubectl/pkg/cmd/get"
)

type JSONPathMatcher struct {
	log logrus.FieldLogger
}

func NewJSONPathMatcher(log logrus.FieldLogger) *JSONPathMatcher {
	return &JSONPathMatcher{log: log}
}

func (j *JSONPathMatcher) IsEventMatchingCriteria(obj json.RawMessage, jsonPath, expValue string) bool {
	if jsonPath == "" {
		return true
	}
	value, err := j.parseJsonpath(obj, jsonPath)
	if err != nil {
		j.log.WithError(err).Errorf("while parsing %s JSONPath", jsonPath)
		return false
	}

	return j.isEqual(expValue, value)
}

func (j *JSONPathMatcher) isEqual(exp, got string) bool {
	// exact match
	if exp == got {
		return true
	}

	// regexp
	matched, err := regexp.MatchString(exp, got)
	if err != nil {
		j.log.WithError(err).Errorf("while matching %q with regex %q", got, exp)
		return false
	}
	return matched
}

func (j *JSONPathMatcher) parseJsonpath(raw []byte, jsonpathStr string) (string, error) {
	fields, err := get.RelaxedJSONPathExpression(jsonpathStr)
	if err != nil {
		return "", err
	}

	jsonPath := jsonpath.New("jsonpath")
	jsonPath.AllowMissingKeys(true)
	if err := jsonPath.Parse(fields); err != nil {
		return "", err
	}

	var obj interface{}
	err = json.Unmarshal(raw, &obj)
	if err != nil {
		return "", err
	}
	values, err := jsonPath.FindResults(obj)
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
