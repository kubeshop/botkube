package format

import (
	"fmt"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// ToTitle returns English specific title casing.
func ToTitle(in fmt.Stringer) string {
	return cases.Title(language.English).String(in.String())
}
