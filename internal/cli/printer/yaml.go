package printer

import (
	"io"

	"sigs.k8s.io/yaml"
)

var _ Printer = &YAML{}

// YAML prints data in YAML format.
type YAML struct{}

// Print marshals input data to YAML format and writes it to a given writer.
func (p *YAML) Print(in interface{}, w io.Writer) error {
	out, err := yaml.Marshal(in)
	if err != nil {
		return err
	}
	_, err = w.Write(out)
	return err
}
