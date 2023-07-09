package helm

import (
	"bytes"
	"fmt"
	"time"

	"helm.sh/helm/v3/pkg/release"
)

// GetStringStatusFromRelease returns release description similar to what Helm does,
// based on https://github.com/helm/helm/blob/f31d4fb3aacabf6102b3ec9214b3433a3dbf1812/cmd/helm/status.go#L126C1-L138C3
func GetStringStatusFromRelease(r *release.Release) string {
	if r == nil {
		return ""
	}

	var buff bytes.Buffer
	buff.WriteString(fmt.Sprintf("NAME: %s\n", r.Name))
	if !r.Info.LastDeployed.IsZero() {
		buff.WriteString(fmt.Sprintf("LAST DEPLOYED: %s\n", r.Info.LastDeployed.Format(time.ANSIC)))
	}
	buff.WriteString(fmt.Sprintf("NAMESPACE: %s\n", r.Namespace))
	buff.WriteString(fmt.Sprintf("STATUS: %s\n", r.Info.Status.String()))
	buff.WriteString(fmt.Sprintf("REVISION: %d\n", r.Version))
	buff.WriteString(fmt.Sprintf("DESCRIPTION: %s\n", r.Info.Description))

	return buff.String()
}
