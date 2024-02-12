package templatex

import (
	"bytes"
	"fmt"
	"strings"
	gotemplate "text/template"
)

const (
	goTplOpeningTag = "{{"
)

// RenderStringIfTemplate renders string if detects a Go template. If not, it returns the same string
func RenderStringIfTemplate(in string, data any) (string, error) {
	if !strings.Contains(in, goTplOpeningTag) {
		return in, nil
	}

	tmpl, err := gotemplate.New("tpl").Parse(in)
	if err != nil {
		return "", fmt.Errorf("while parsing template %q: %w", in, err)
	}

	var result bytes.Buffer
	err = tmpl.Execute(&result, data)
	if err != nil {
		return "", fmt.Errorf("while rendering string %q: %w", in, err)
	}

	return result.String(), nil
}
