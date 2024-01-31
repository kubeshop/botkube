package httpx

import (
	"strings"
)

func CanonicalURLPath(path string) string {
	normalized := strings.TrimRight(path, "/")
	if !strings.HasSuffix(normalized, "/") {
		normalized += "/"
	}
	return normalized
}
