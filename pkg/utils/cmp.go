package utils

import (
	"fmt"
	"strings"

	"github.com/google/go-cmp/cmp"
	"github.com/infracloudio/botkube/pkg/config"
)

// DiffReporter is a simple custom reporter that records differences
// detected in Object during comparison.
type DiffReporter struct {
	field string
	path  cmp.Path
	diffs []string
}

// PushStep custom implements Reporter interface
func (r *DiffReporter) PushStep(ps cmp.PathStep) {
	r.path = append(r.path, ps)
}

// Report custom implements Reporter interface
func (r *DiffReporter) Report(rs cmp.Result) {
	if !rs.Equal() {
		vx, vy := r.path.Last().Values()
		path := fmt.Sprintf("%#v", r.path)
		if ok := strings.Contains(path, "."+strings.Title(r.field)); ok {
			r.diffs = append(r.diffs, fmt.Sprintf("%#v:\n\t-: %+v\n\t+: %+v\n", r.path, vx, vy))
		}
	}
}

// PopStep custom implements Reporter interface
func (r *DiffReporter) PopStep() {
	r.path = r.path[:len(r.path)-1]
}

// String custom implements Reporter interface
func (r *DiffReporter) String() string {
	return strings.Join(r.diffs, "\n")
}

// Diff provides differences between two objects spec
func Diff(x, y interface{}, updatesetting config.UpdateSetting) string {
	msg := ""
	for _, val := range updatesetting.Fields {
		var r DiffReporter
		r.field = string(val)
		cmp.Equal(x, y, cmp.Reporter(&r))
		msg = msg + r.String()
	}
	return msg
}
