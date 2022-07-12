package d

import (
	"fmt"
	"regexp"
	"strings"
	"testing"
)

func Test(t *testing.T) {
	valuesSectionBuilder := strings.Builder{}
	valuesSectionBuilder.WriteString(`{{ define "chart.maintainersTable" }}`)
	valuesSectionBuilder.WriteString("| Name | Email | Url |\n")
	valuesSectionBuilder.WriteString("| ---- | ------ | --- |\n")
	valuesSectionBuilder.WriteString("  {{- range .Maintainers }}")
	valuesSectionBuilder.WriteString("\n| {{ .Name }} | {{ if .Email }}<{{ .Email }}>{{ end }} | {{ if .Url }}<{{ .Url }}>{{ end }} |")
	valuesSectionBuilder.WriteString("  {{- end }}")
	valuesSectionBuilder.WriteString("{{ end }}")
	print(valuesSectionBuilder.String())
}

func Test2(t *testing.T) {
	in := `A secret containing a kubeconfig to use. # Secret format: #  data: #    config: {base64_encoded_kubeconfig}`

	d := regexp.MustCompile(`#.*`)
	fmt.Println(d.ReplaceAllString(in, ""))

}
