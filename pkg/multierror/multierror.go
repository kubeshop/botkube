package multierror

import (
	"fmt"
	"strings"

	"github.com/hashicorp/go-multierror"
)

// New create new *multierror.Error with formatting without `\n\n` at the end
func New() *multierror.Error {
	return &multierror.Error{
		ErrorFormat: listFormatFunc,
	}
}

// Append is a wrapper for multierror.Append function.
func Append(err error, errs ...error) *multierror.Error {
	return multierror.Append(err, errs...)
}

// listFormatFunc is a basic formatter that outputs the number of errors
// that occurred along with a bullet point list of the errors.
//
// it's a copy of https://github.com/hashicorp/go-multierror/blob/9974e9ec57696378079ecc3accd3d6f29401b3a0/format.go#L14
// with removed additional two new lines (`\n\n`) added to error message.
func listFormatFunc(es []error) string {
	if len(es) == 1 {
		return fmt.Sprintf("1 error occurred:\n\t* %s", es[0])
	}

	points := make([]string, len(es))
	for i, err := range es {
		points[i] = fmt.Sprintf("* %s", err)
	}

	return fmt.Sprintf(
		"%d errors occurred:\n\t%s",
		len(es), strings.Join(points, "\n\t"))
}
