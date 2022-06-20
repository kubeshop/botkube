package filterengine

import (
	"bytes"
	"fmt"
	"testing"
	"text/tabwriter"

	logtest "github.com/sirupsen/logrus/hooks/test"

	"github.com/infracloudio/botkube/pkg/config"
)

func TestFoo(t *testing.T) {
	buf := new(bytes.Buffer)
	w := tabwriter.NewWriter(buf, 5, 0, 1, ' ', 0)

	log, _ := logtest.NewNullLogger()
	fmt.Fprintln(w, "FILTER\tENABLED\tDESCRIPTION")
	for k, v := range WithAllFilters(log, nil, nil, &config.Config{}).ShowFilters() {
		fmt.Fprintf(w, "%s\t%v\t%s\n", k.Name(), v, k.Describe())
	}

	w.Flush()
	fmt.Println(buf.String())
}
