package helm

import (
	"fmt"
	"time"

	"github.com/muesli/reflow/indent"
	"go.szostok.io/version/style"
	"helm.sh/helm/v3/pkg/release"

	"github.com/kubeshop/botkube/internal/cli/printer"
)

var releaseGoTpl = `
  {{ Key "Name"           }}    {{ .Name                        | Val }}
  {{ Key "Namespace"      }}    {{ .Namespace                   | Val }}
  {{ Key "Last Deployed"  }}    {{ .LastDeployed  | FmtDate     | Val }}
  {{ Key "Revision"       }}    {{ .Revision                    | Val }}
  {{ Key "Description"    }}    {{ .Description                 | Val }}
`

// PrintReleaseStatus returns release description similar to what Helm does,
// based on https://github.com/helm/helm/blob/f31d4fb3aacabf6102b3ec9214b3433a3dbf1812/cmd/helm/status.go#L126C1-L138C3
func PrintReleaseStatus(status *printer.StatusPrinter, r *release.Release) error {
	if r == nil {
		return nil
	}

	renderer := style.NewGoTemplateRender(style.DefaultConfig(releaseGoTpl))

	properties := make(map[string]string)
	properties["Name"] = r.Name
	if !r.Info.LastDeployed.IsZero() {
		properties["LastDeployed"] = r.Info.LastDeployed.Format(time.ANSIC)
	}
	properties["Namespace"] = r.Namespace
	properties["Status"] = r.Info.Status.String()
	properties["Revision"] = fmt.Sprintf("%d", r.Version)
	properties["Description"] = r.Info.Description

	desc, err := renderer.Render(properties, true)
	if err != nil {
		return err
	}

	status.InfoWithBody("Release details:", indent.String(desc, 2))
	return nil
}
