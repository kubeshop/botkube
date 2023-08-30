package argocd

import (
	"bytes"
	"fmt"
	"strings"
	gotemplate "text/template"

	"github.com/kubeshop/botkube/pkg/api/source"
)

const (
	goTplOpeningTag = "{{"
)

func renderStringIfTemplate(tpl string, srcCtx source.CommonSourceContext) (string, error) {
	if !strings.Contains(tpl, goTplOpeningTag) {
		return tpl, nil
	}

	tmpl, err := gotemplate.New("tpl").Parse(tpl)
	if err != nil {
		return "", err
	}

	var result bytes.Buffer
	err = tmpl.Execute(&result, srcCtx)
	if err != nil {
		return "", fmt.Errorf("while rendering string %q: %w", tpl, err)
	}

	return result.String(), nil
}
