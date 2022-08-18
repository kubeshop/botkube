package utils

import (
	"strings"
)

func Intersect(this, that []string) bool {
	for _, i := range this {
		for _, j := range that {
			if strings.ToLower(i) == strings.ToLower(j) {
				return true
			}
		}
	}
	return false
}
