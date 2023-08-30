package printer

import (
	"io"

	"github.com/hokaccha/go-prettyjson"
)

var _ Printer = &JSON{}

// JSON prints data in JSON format.
type JSON struct{}

// Print marshals input data to JSON format and writes it to a given writer.
func (p *JSON) Print(in interface{}, w io.Writer) error {
	out, err := prettyjson.Marshal(in)
	if err != nil {
		return err
	}

	_, err = w.Write(out)
	return err
}
