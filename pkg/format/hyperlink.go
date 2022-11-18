package format

import (
	"regexp"
	"strings"
)

const hyperlinkRegex = `(?m)<https?:\/\/[a-z.0-9\/\-_=]*\|([a-z.0-9\/\-_=]*)>`

// RemoveHyperlinks removes the hyperlink text from url
func RemoveHyperlinks(in string) string {
	compiledRegex := regexp.MustCompile(hyperlinkRegex)
	matched := compiledRegex.FindAllStringSubmatch(in, -1)
	if len(matched) >= 1 {
		for _, match := range matched {
			if len(match) == 2 {
				in = strings.ReplaceAll(in, match[0], match[1])
			}
		}
	}
	return in
}
