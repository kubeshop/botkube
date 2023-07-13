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
  {{ Key "Version"        }}    {{ .Version                     | Val }}
  {{ Key "Last Deployed"  }}    {{ .LastDeployed  | FmtDate     | Val }}
  {{ Key "Revision"       }}    {{ .Revision                    | Val }}
`

// PrintReleaseStatus returns release description similar to what Helm does,
// based on https://github.com/helm/helm/blob/f31d4fb3aacabf6102b3ec9214b3433a3dbf1812/cmd/helm/status.go#L126C1-L138C3
func PrintReleaseStatus(header string, status *printer.StatusPrinter, r *release.Release) error {
	if r == nil {
		return nil
	}

	renderer := style.NewGoTemplateRender(style.DefaultConfig(releaseGoTpl))

	properties := make(map[string]string)
	properties["Name"] = r.Name
	properties["Namespace"] = r.Namespace
	properties["Revision"] = fmt.Sprintf("%d", r.Version)

	if r.Info != nil {
		if !r.Info.LastDeployed.IsZero() {
			properties["LastDeployed"] = r.Info.LastDeployed.Format(time.ANSIC)
		}
		properties["Status"] = r.Info.Status.String()
		properties["Description"] = r.Info.Description
	}

	if r.Chart != nil {
		properties["Version"] = r.Chart.AppVersion()
	}

	desc, err := renderer.Render(properties, true)
	if err != nil {
		return err
	}

	status.InfoWithBody(header, indent.String(desc, 2))
	return nil
}
