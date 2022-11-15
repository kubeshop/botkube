package execute

import (
	"bytes"
	"fmt"
	"text/tabwriter"

	"github.com/kubeshop/botkube/pkg/config"
)

type ActionManager struct {
	actions map[string]bool
}

func NewActionManager(cfg config.Actions) *ActionManager {
	a := make(map[string]bool)
	for k, v := range cfg {
		a[k] = v.Enabled
	}
	return &ActionManager{a}
}

func (a *ActionManager) listActions() map[string]bool {
	out := make(map[string]bool)
	for k, v := range a.actions {
		out[k] = v
	}
	return out
}

func (a *ActionManager) tabularOutput() string {
	buf := new(bytes.Buffer)
	w := tabwriter.NewWriter(buf, 5, 0, 1, ' ', 0)
	fmt.Fprintln(w, "ACTION\tENABLED")
	for name, enabled := range a.listActions() {
		fmt.Fprintf(w, "%s\t%v\n", name, enabled)
	}
	w.Flush()
	return buf.String()
}

func (a *ActionManager) enableAction(name string) bool {
	if _, ok := a.actions[name]; ok {
		a.actions[name] = true
		return true
	}
	return false
}

func (a *ActionManager) disableAction(name string) bool {
	if _, ok := a.actions[name]; ok {
		a.actions[name] = false
		return true
	}
	return false
}
