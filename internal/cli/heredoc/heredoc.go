package heredoc

import (
	"strings"

	"github.com/MakeNowJust/heredoc"
)

const cliTag = "<cli>"

// WithCLIName returns unindented and formatted string as here-document.
// Replace all <cli> with a given name.
func WithCLIName(raw string, cliName string) string {
	return strings.ReplaceAll(heredoc.Doc(raw), cliTag, cliName)
}

// Docf returns unindented and formatted string as here-document.
func Docf(raw string, args ...interface{}) string {
	return heredoc.Docf(raw, args...)
}
